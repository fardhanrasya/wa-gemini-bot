package poker

import (
	"errors"
	"fmt"
)

// ==========================================================================
// Game — state machine untuk satu permainan Texas Hold'em Poker.
//
// Desain: Game adalah "deep module" yang menyembunyikan semua kompleksitas
// aturan poker di balik interface yang sederhana. Caller (PokerService)
// hanya perlu memanggil AddPlayer, StartRound, HandleAction, dan membaca
// hasilnya. Game tidak tahu tentang WhatsApp, timer, atau goroutine —
// semua itu dikelola oleh layer di atasnya.
//
// State machine: Lobby → PreFlop → Flop → Turn → River → Showdown
// Transisi antar state terjadi otomatis saat semua player selesai bertindak.
// ==========================================================================

// GamePhase merepresentasikan fase permainan saat ini.
type GamePhase int

const (
	PhaseLobby    GamePhase = iota // Menunggu player join
	PhasePreFlop                   // Kartu hole dibagikan, ronde taruhan pertama
	PhaseFlop                      // 3 community cards dibuka
	PhaseTurn                      // Kartu ke-4 dibuka
	PhaseRiver                     // Kartu ke-5 dibuka
	PhaseShowdown                  // Penentuan pemenang
	PhaseFinished                  // Game selesai (satu pemenang tersisa)
)

var phaseNames = map[GamePhase]string{
	PhaseLobby:    "Lobby",
	PhasePreFlop:  "Pre-Flop",
	PhaseFlop:     "Flop",
	PhaseTurn:     "Turn",
	PhaseRiver:    "River",
	PhaseShowdown: "Showdown",
	PhaseFinished: "Finished",
}

func (p GamePhase) String() string { return phaseNames[p] }

// PlayerStatus merepresentasikan status seorang player dalam ronde ini.
type PlayerStatus int

const (
	StatusActive     PlayerStatus = iota // Masih bermain
	StatusFolded                         // Sudah fold ronde ini
	StatusAllIn                          // Sudah all-in
	StatusEliminated                     // Kehabisan chip (keluar dari game)
)

// Player merepresentasikan seorang pemain poker.
type Player struct {
	Name       string
	JID        string // WhatsApp JID untuk DM
	Chips      int
	BuyIn      int // Jumlah chip yang dibawa saat join (untuk tracking change)
	HoleCards  [2]Card
	Status     PlayerStatus
	CurrentBet int // Taruhan di ronde betting saat ini
	TotalBet   int // Total taruhan di seluruh ronde (untuk pot calculation)
}

// ActionType merepresentasikan jenis aksi yang bisa dilakukan player.
type ActionType int

const (
	ActionFold ActionType = iota
	ActionCheck
	ActionCall
	ActionBet
	ActionRaise
	ActionAllIn
)

// Action merepresentasikan satu aksi dari player.
type Action struct {
	Type   ActionType
	Amount int // Untuk Bet/Raise: jumlah yang ditaruhkan
}

// ActionResult adalah output dari HandleAction — berisi semua info yang
// dibutuhkan oleh PokerService untuk mengirim pesan yang tepat.
type ActionResult struct {
	Valid           bool
	Message         string // Pesan untuk dikirim ke grup
	NextPlayer      string // Nama player berikutnya (kosong jika phase selesai)
	PhaseEnded      bool   // True jika betting round selesai → advance phase
	NewPhase        GamePhase
	CommunityCards  []Card                 // Community cards baru (saat phase advance)
	ShowdownResults []ShowdownPlayerResult // Hanya diisi saat showdown
	RoundOver       bool                   // True jika ronde selesai (showdown atau semua fold)
	Winners         []WinResult
	GameOver        bool   // True jika hanya 1 player tersisa
	FinalWinner     string // Nama pemenang game (hanya saat GameOver)
}

// ShowdownPlayerResult berisi hasil evaluasi kartu seorang player saat showdown.
type ShowdownPlayerResult struct {
	Name      string
	HoleCards [2]Card
	BestHand  HandResult
}

// WinResult berisi info pemenang satu pot.
type WinResult struct {
	PlayerName string
	Amount     int
	HandDesc   string // Deskripsi hand yang menang (kosong jika menang via fold)
}

// Game mengelola seluruh state permainan poker.
type Game struct {
	Players          []*Player
	Phase            GamePhase
	Deck             *Deck
	CommunityCards   []Card
	DealerIndex      int // Posisi dealer button
	SmallBlind       int
	BigBlind         int
	CurrentBetAmount int // Taruhan tertinggi saat ini di ronde betting ini

	// Tracking giliran — index ke Players slice
	currentTurnIndex int
	// firstActorIndex — siapa yang pertama kali bertindak di ronde ini
	// (untuk mendeteksi kapan semua player sudah bertindak)
	lastRaiserIndex  int
	actionsThisRound int

	// Config
	startingChips int
}

// NewGame membuat Game baru dalam fase Lobby.
func NewGame(smallBlind, bigBlind, startingChips int) *Game {
	return &Game{
		Phase:         PhaseLobby,
		SmallBlind:    smallBlind,
		BigBlind:      bigBlind,
		startingChips: startingChips,
		DealerIndex:   -1, // Akan diset saat StartRound
	}
}

// ==========================================================================
// Lobby management
// ==========================================================================

const (
	MinPlayers = 2
	MaxPlayers = 8
)

// AddPlayer menambahkan player ke lobby.
func (g *Game) AddPlayer(name, jid string) error {
	if g.Phase != PhaseLobby {
		return errors.New("tidak bisa join — game sudah berjalan")
	}
	if len(g.Players) >= MaxPlayers {
		return fmt.Errorf("lobby penuh (max %d pemain)", MaxPlayers)
	}
	for _, p := range g.Players {
		if p.Name == name {
			return errors.New("kamu sudah bergabung")
		}
	}

	g.Players = append(g.Players, &Player{
		Name:  name,
		JID:   jid,
		Chips: g.startingChips,
	})
	return nil
}

// PlayerCount mengembalikan jumlah player di lobby/game.
func (g *Game) PlayerCount() int {
	return len(g.Players)
}

// ==========================================================================
// Round management — memulai ronde baru
// ==========================================================================

// RoundStartInfo berisi data yang dibutuhkan untuk mengirim pesan awal ronde.
type RoundStartInfo struct {
	DealerName     string
	SmallBlindName string
	BigBlindName   string
	SmallBlindAmt  int
	BigBlindAmt    int
	Pot            int
	Players        []RoundPlayerInfo
	FirstTurnName  string
}

// RoundPlayerInfo berisi info player untuk ditampilkan di awal ronde.
type RoundPlayerInfo struct {
	Name      string
	JID       string
	Chips     int
	HoleCards [2]Card
}

// StartRound memulai ronde baru: shuffle deck, deal kartu, post blinds.
// Hanya bisa dipanggil saat Phase == PhaseLobby atau setelah ronde sebelumnya selesai.
func (g *Game) StartRound() (*RoundStartInfo, error) {
	activePlayers := g.activePlayers()
	if len(activePlayers) < MinPlayers {
		return nil, fmt.Errorf("butuh minimal %d pemain aktif", MinPlayers)
	}

	// Advance dealer button
	g.DealerIndex = g.nextActivePlayerIndex(g.DealerIndex)

	// Reset state untuk ronde baru
	g.Deck = NewDeck()
	g.CommunityCards = nil
	g.CurrentBetAmount = 0

	// Reset semua player aktif
	for _, p := range g.Players {
		if p.Status == StatusEliminated {
			continue
		}
		p.Status = StatusActive
		p.CurrentBet = 0
		p.TotalBet = 0
		p.HoleCards = [2]Card{}
	}

	// Deal hole cards
	for i := 0; i < 2; i++ {
		for _, p := range g.Players {
			if p.Status == StatusEliminated {
				continue
			}
			p.HoleCards[i] = g.Deck.Draw()
		}
	}

	// Post blinds
	sbIndex := g.nextActivePlayerIndex(g.DealerIndex)
	bbIndex := g.nextActivePlayerIndex(sbIndex)

	sbPlayer := g.Players[sbIndex]
	bbPlayer := g.Players[bbIndex]

	g.postBlind(sbPlayer, g.SmallBlind)
	g.postBlind(bbPlayer, g.BigBlind)

	g.CurrentBetAmount = g.BigBlind
	g.Phase = PhasePreFlop

	// Pre-Flop: aksi dimulai dari player setelah big blind
	g.currentTurnIndex = g.nextActivePlayerIndex(bbIndex)
	g.lastRaiserIndex = bbIndex // BB dianggap "raiser" awal
	g.actionsThisRound = 0

	// Build info
	var playerInfos []RoundPlayerInfo
	for _, p := range g.Players {
		if p.Status == StatusEliminated {
			continue
		}
		playerInfos = append(playerInfos, RoundPlayerInfo{
			Name:      p.Name,
			JID:       p.JID,
			Chips:     p.Chips,
			HoleCards: p.HoleCards,
		})
	}

	return &RoundStartInfo{
		DealerName:     g.Players[g.DealerIndex].Name,
		SmallBlindName: sbPlayer.Name,
		BigBlindName:   bbPlayer.Name,
		SmallBlindAmt:  sbPlayer.CurrentBet,
		BigBlindAmt:    bbPlayer.CurrentBet,
		Pot:            g.totalPot(),
		Players:        playerInfos,
		FirstTurnName:  g.Players[g.currentTurnIndex].Name,
	}, nil
}

func (g *Game) postBlind(p *Player, amount int) {
	if amount > p.Chips {
		amount = p.Chips
		p.Status = StatusAllIn
	}
	p.Chips -= amount
	p.CurrentBet = amount
	p.TotalBet = amount
}

// ==========================================================================
// Action handling — core state machine logic
// ==========================================================================

// GetCurrentTurnPlayer mengembalikan nama player yang gilirannya saat ini.
// Return kosong jika tidak ada (fase lobby/showdown/finished).
func (g *Game) GetCurrentTurnPlayer() string {
	if g.Phase == PhaseLobby || g.Phase == PhaseShowdown || g.Phase == PhaseFinished {
		return ""
	}
	if g.currentTurnIndex < 0 || g.currentTurnIndex >= len(g.Players) {
		return ""
	}
	return g.Players[g.currentTurnIndex].Name
}

// GetCurrentBet mengembalikan taruhan tertinggi saat ini.
func (g *Game) GetCurrentBet() int {
	return g.CurrentBetAmount
}

// GetPot mengembalikan total chip di pot saat ini.
func (g *Game) GetPot() int {
	return g.totalPot()
}

// GetPlayerChips mengembalikan jumlah chip seorang player.
func (g *Game) GetPlayerChips(name string) int {
	for _, p := range g.Players {
		if p.Name == name {
			return p.Chips
		}
	}
	return 0
}

// GetPlayerCurrentBet mengembalikan taruhan seorang player di ronde betting ini.
func (g *Game) GetPlayerCurrentBet(name string) int {
	for _, p := range g.Players {
		if p.Name == name {
			return p.CurrentBet
		}
	}
	return 0
}

// HandleAction memproses aksi dari seorang player.
// Ini adalah method paling "deep" — menyembunyikan semua validasi, transisi state,
// dan kalkulasi dari caller.
func (g *Game) HandleAction(playerName string, action Action) ActionResult {
	// Guard: pastikan game aktif dan ini giliran player yang benar
	currentPlayer := g.GetCurrentTurnPlayer()
	if currentPlayer == "" {
		return ActionResult{Message: "Tidak ada game aktif saat ini."}
	}
	if playerName != currentPlayer {
		return ActionResult{Message: fmt.Sprintf("Bukan giliranmu! Sekarang giliran %s.", currentPlayer)}
	}

	player := g.getPlayer(playerName)
	if player == nil {
		return ActionResult{Message: "Player tidak ditemukan."}
	}

	// Proses aksi
	var result ActionResult
	switch action.Type {
	case ActionFold:
		result = g.handleFold(player)
	case ActionCheck:
		result = g.handleCheck(player)
	case ActionCall:
		result = g.handleCall(player)
	case ActionBet:
		result = g.handleBet(player, action.Amount)
	case ActionRaise:
		result = g.handleRaise(player, action.Amount)
	case ActionAllIn:
		result = g.handleAllIn(player)
	default:
		return ActionResult{Message: "Aksi tidak dikenal."}
	}

	if !result.Valid {
		return result
	}

	// Cek apakah hanya 1 player aktif tersisa (semua lain fold)
	activePlayers := g.playersInHand()
	if len(activePlayers) == 1 {
		return g.winByFold(activePlayers[0])
	}

	// Cek apakah ronde betting selesai
	if g.isBettingRoundComplete() {
		return g.advancePhase(result)
	}

	// Lanjut ke player berikutnya
	g.advanceToNextPlayer()
	result.NextPlayer = g.GetCurrentTurnPlayer()
	return result
}

// ==========================================================================
// Individual action handlers
// ==========================================================================

func (g *Game) handleFold(p *Player) ActionResult {
	p.Status = StatusFolded
	g.actionsThisRound++
	return ActionResult{
		Valid:   true,
		Message: fmt.Sprintf("🏳️ %s fold.", p.Name),
	}
}

func (g *Game) handleCheck(p *Player) ActionResult {
	// Check hanya bisa dilakukan jika taruhan saat ini = taruhan player
	if g.CurrentBetAmount > p.CurrentBet {
		callAmount := g.CurrentBetAmount - p.CurrentBet
		return ActionResult{
			Valid:   false,
			Message: fmt.Sprintf("Tidak bisa check — ada taruhan %d 💰. Kamu perlu call %d atau fold.", g.CurrentBetAmount, callAmount),
		}
	}
	g.actionsThisRound++
	return ActionResult{
		Valid:   true,
		Message: fmt.Sprintf("☑️ %s check.", p.Name),
	}
}

func (g *Game) handleCall(p *Player) ActionResult {
	callAmount := g.CurrentBetAmount - p.CurrentBet
	if callAmount <= 0 {
		return ActionResult{
			Valid:   false,
			Message: "Tidak ada taruhan untuk di-call. Gunakan check atau bet.",
		}
	}

	if callAmount >= p.Chips {
		// Auto all-in jika chip tidak cukup
		return g.handleAllIn(p)
	}

	p.Chips -= callAmount
	p.CurrentBet += callAmount
	p.TotalBet += callAmount
	g.actionsThisRound++

	return ActionResult{
		Valid:   true,
		Message: fmt.Sprintf("✅ %s call %d 💰 (sisa: %d 💰)", p.Name, callAmount, p.Chips),
	}
}

func (g *Game) handleBet(p *Player, amount int) ActionResult {
	// Bet hanya bisa dilakukan jika belum ada taruhan
	if g.CurrentBetAmount > 0 && g.Phase != PhasePreFlop {
		return ActionResult{
			Valid:   false,
			Message: "Sudah ada taruhan — gunakan raise atau call.",
		}
	}
	// Pre-flop: selalu ada taruhan (big blind), jadi gunakan raise
	if g.Phase == PhasePreFlop {
		return g.handleRaise(p, amount)
	}

	if amount < g.BigBlind {
		return ActionResult{
			Valid:   false,
			Message: fmt.Sprintf("Bet minimum: %d 💰", g.BigBlind),
		}
	}

	if amount >= p.Chips {
		return g.handleAllIn(p)
	}

	p.Chips -= amount
	p.CurrentBet = amount
	p.TotalBet += amount
	g.CurrentBetAmount = amount
	g.lastRaiserIndex = g.currentTurnIndex
	g.actionsThisRound = 1 // Reset — semua player lain harus merespons

	return ActionResult{
		Valid:   true,
		Message: fmt.Sprintf("💰 %s bet %d 💰 (sisa: %d 💰)", p.Name, amount, p.Chips),
	}
}

func (g *Game) handleRaise(p *Player, totalAmount int) ActionResult {
	if g.CurrentBetAmount == 0 && g.Phase != PhasePreFlop {
		return ActionResult{
			Valid:   false,
			Message: "Belum ada taruhan — gunakan bet.",
		}
	}

	minRaise := g.CurrentBetAmount * 2
	if g.Phase == PhasePreFlop && g.CurrentBetAmount == g.BigBlind {
		minRaise = g.BigBlind * 2
	}

	if totalAmount < minRaise {
		return ActionResult{
			Valid:   false,
			Message: fmt.Sprintf("Raise minimum ke %d 💰 (2x taruhan saat ini %d).", minRaise, g.CurrentBetAmount),
		}
	}

	raiseAmount := totalAmount - p.CurrentBet
	if raiseAmount >= p.Chips {
		return g.handleAllIn(p)
	}

	p.Chips -= raiseAmount
	p.CurrentBet = totalAmount
	p.TotalBet += raiseAmount
	g.CurrentBetAmount = totalAmount
	g.lastRaiserIndex = g.currentTurnIndex
	g.actionsThisRound = 1 // Reset — semua player lain harus merespons

	return ActionResult{
		Valid:   true,
		Message: fmt.Sprintf("⬆️ %s raise ke %d 💰 (sisa: %d 💰)", p.Name, totalAmount, p.Chips),
	}
}

func (g *Game) handleAllIn(p *Player) ActionResult {
	allInAmount := p.Chips
	p.CurrentBet += allInAmount
	p.TotalBet += allInAmount
	p.Chips = 0
	p.Status = StatusAllIn

	if p.CurrentBet > g.CurrentBetAmount {
		g.CurrentBetAmount = p.CurrentBet
		g.lastRaiserIndex = g.currentTurnIndex
		g.actionsThisRound = 1
	} else {
		g.actionsThisRound++
	}

	return ActionResult{
		Valid:   true,
		Message: fmt.Sprintf("🔥 %s ALL IN! %d 💰", p.Name, p.CurrentBet),
	}
}

// ==========================================================================
// State transition — advancing phases
// ==========================================================================

func (g *Game) advancePhase(prevResult ActionResult) ActionResult {
	result := prevResult

	// Reset betting state untuk ronde baru
	for _, p := range g.Players {
		p.CurrentBet = 0
	}
	g.CurrentBetAmount = 0
	g.actionsThisRound = 0

	switch g.Phase {
	case PhasePreFlop:
		g.Phase = PhaseFlop
		g.Deck.Burn()
		g.CommunityCards = append(g.CommunityCards, g.Deck.Draw(), g.Deck.Draw(), g.Deck.Draw())
		result.NewPhase = PhaseFlop

	case PhaseFlop:
		g.Phase = PhaseTurn
		g.Deck.Burn()
		g.CommunityCards = append(g.CommunityCards, g.Deck.Draw())
		result.NewPhase = PhaseTurn

	case PhaseTurn:
		g.Phase = PhaseRiver
		g.Deck.Burn()
		g.CommunityCards = append(g.CommunityCards, g.Deck.Draw())
		result.NewPhase = PhaseRiver

	case PhaseRiver:
		g.Phase = PhaseShowdown
		result.NewPhase = PhaseShowdown
		return g.resolveShowdown(result)
	}

	result.PhaseEnded = true
	result.CommunityCards = g.CommunityCards

	// Cek apakah semua remaining player sudah all-in — langsung ke showdown
	canAct := g.playersWhoCanAct()
	if len(canAct) <= 1 {
		// Sisa 0 atau 1 player yang bisa bertindak — deal remaining cards dan showdown
		return g.runOutCommunityCards(result)
	}

	// Set giliran pertama di ronde baru: player aktif pertama setelah dealer
	g.currentTurnIndex = g.nextPlayerWhoCanActFrom(g.DealerIndex)
	g.lastRaiserIndex = -1
	result.NextPlayer = g.GetCurrentTurnPlayer()
	return result
}

// runOutCommunityCards membuka sisa community cards dan langsung showdown.
// Terjadi saat semua player sudah all-in.
func (g *Game) runOutCommunityCards(prevResult ActionResult) ActionResult {
	result := prevResult
	for len(g.CommunityCards) < 5 {
		g.Deck.Burn()
		g.CommunityCards = append(g.CommunityCards, g.Deck.Draw())
	}
	result.CommunityCards = g.CommunityCards
	g.Phase = PhaseShowdown
	result.NewPhase = PhaseShowdown
	return g.resolveShowdown(result)
}

// resolveShowdown mengevaluasi semua tangan dan menentukan pemenang.
func (g *Game) resolveShowdown(prevResult ActionResult) ActionResult {
	result := prevResult
	result.RoundOver = true
	result.CommunityCards = g.CommunityCards // Selalu sertakan community cards di showdown

	// Evaluasi hand setiap player yang masih in hand
	var showdownResults []ShowdownPlayerResult
	for _, p := range g.playersInHand() {
		allCards := make([]Card, 0, 7)
		allCards = append(allCards, p.HoleCards[0], p.HoleCards[1])
		allCards = append(allCards, g.CommunityCards...)
		hand := EvaluateBestHand(allCards)

		showdownResults = append(showdownResults, ShowdownPlayerResult{
			Name:      p.Name,
			HoleCards: p.HoleCards,
			BestHand:  hand,
		})
	}
	result.ShowdownResults = showdownResults

	// Hitung pot dan distribusikan
	result.Winners = g.distributePots(showdownResults)

	// Cek eliminasi dan game over
	g.eliminateBrokePlayers()
	if g.isGameOver() {
		result.GameOver = true
		for _, p := range g.Players {
			if p.Status != StatusEliminated {
				result.FinalWinner = p.Name
				g.Phase = PhaseFinished
				break
			}
		}
	}

	return result
}

// winByFold — semua player fold kecuali satu.
func (g *Game) winByFold(winner *Player) ActionResult {
	pot := g.totalPot()
	winner.Chips += pot

	result := ActionResult{
		Valid:     true,
		RoundOver: true,
		Winners: []WinResult{{
			PlayerName: winner.Name,
			Amount:     pot,
		}},
		Message: fmt.Sprintf("🏆 Semua pemain fold!\n%s menang %d 💰 tanpa perlu buka kartu! 🎉", winner.Name, pot),
	}

	// Cek eliminasi dan game over
	g.eliminateBrokePlayers()
	if g.isGameOver() {
		result.GameOver = true
		result.FinalWinner = winner.Name
		g.Phase = PhaseFinished
	}

	return result
}

// distributePots menghitung pemenang untuk setiap pot (termasuk side pot).
func (g *Game) distributePots(showdownResults []ShowdownPlayerResult) []WinResult {
	// Build contributions
	var contributions []PlayerContribution
	for _, p := range g.Players {
		if p.Status == StatusEliminated {
			continue
		}
		contributions = append(contributions, PlayerContribution{
			Name:   p.Name,
			Amount: p.TotalBet,
			Folded: p.Status == StatusFolded,
		})
	}

	pots := CalculatePots(contributions)
	var winners []WinResult

	for _, pot := range pots {
		if len(pot.EligiblePlayers) == 0 {
			continue
		}

		// Cari hand terbaik di antara eligible players
		var bestResults []*ShowdownPlayerResult
		for i := range showdownResults {
			for _, eligible := range pot.EligiblePlayers {
				if showdownResults[i].Name == eligible {
					if len(bestResults) == 0 {
						bestResults = append(bestResults, &showdownResults[i])
					} else {
						cmp := CompareHands(showdownResults[i].BestHand, bestResults[0].BestHand)
						if cmp > 0 {
							bestResults = []*ShowdownPlayerResult{&showdownResults[i]}
						} else if cmp == 0 {
							bestResults = append(bestResults, &showdownResults[i])
						}
					}
				}
			}
		}

		if len(bestResults) > 0 {
			splitAmount := pot.Amount / len(bestResults)
			remainder := pot.Amount % len(bestResults)

			for i, bestResult := range bestResults {
				amountToGive := splitAmount
				if i == 0 {
					amountToGive += remainder // Sisa chip ganjil diberikan ke pemenang pertama
				}

				// Berikan chip ke pemenang
				for _, p := range g.Players {
					if p.Name == bestResult.Name {
						p.Chips += amountToGive
						break
					}
				}
				winners = append(winners, WinResult{
					PlayerName: bestResult.Name,
					Amount:     amountToGive,
					HandDesc:   bestResult.BestHand.Description,
				})
			}
		}
	}

	return winners
}

// ==========================================================================
// Game status & info
// ==========================================================================

// ChipStandings mengembalikan daftar player dan chip mereka, terurut descending.
type ChipStanding struct {
	Name   string
	Chips  int
	Change int // Perubahan chip dari awal ronde
	Status PlayerStatus
}

// GetChipStandings mengembalikan standings semua player yang masih aktif (tidak eliminated).
func (g *Game) GetChipStandings() []ChipStanding {
	var standings []ChipStanding
	for _, p := range g.Players {
		// Skip player yang sudah eliminated (sudah leave atau bangkrut)
		if p.Status == StatusEliminated {
			continue
		}
		// Hitung change berdasarkan buy-in aktual, bukan startingChips default
		change := p.Chips - p.BuyIn
		standings = append(standings, ChipStanding{
			Name:   p.Name,
			Chips:  p.Chips,
			Change: change,
			Status: p.Status,
		})
	}
	// Sort by chips descending
	for i := 0; i < len(standings); i++ {
		for j := i + 1; j < len(standings); j++ {
			if standings[j].Chips > standings[i].Chips {
				standings[i], standings[j] = standings[j], standings[i]
			}
		}
	}
	return standings
}

// PrepareNextRound menyiapkan game untuk ronde berikutnya.
// Reset fase ke Lobby-like state tanpa mengubah player list atau chip.
func (g *Game) PrepareNextRound() {
	g.Phase = PhaseLobby
	g.CommunityCards = nil
	g.CurrentBetAmount = 0
}

// ==========================================================================
// Internal helpers
// ==========================================================================

// RefundAll mengembalikan map berisi jumlah chip yang harus dikembalikan ke
// tiap pemain jika game mendadak berhenti.
// Perhitungannya: hanya sisa Chips di meja (TotalBet sudah masuk pot dan tidak di-refund).
func (g *Game) RefundAll() map[string]int {
	refunds := make(map[string]int)
	for _, p := range g.Players {
		// Hanya refund chip yang masih di tangan player, bukan yang sudah di pot
		if p.Chips > 0 {
			refunds[p.Name] = p.Chips
		}
	}
	return refunds
}

// Leave memproses pemain yang keluar dari meja.
// Jika game sedang berjalan, pemain otomatis fold.
// Mengembalikan sisa chip (p.Chips), flag found, dan opsi ActionResult jika ada perubahan state.
func (g *Game) Leave(playerName string) (int, bool, *ActionResult) {
	var p *Player
	var pIndex int
	for i, player := range g.Players {
		if player.Name == playerName {
			p = player
			pIndex = i
			break
		}
	}
	if p == nil {
		return 0, false, nil
	}

	remaining := p.Chips
	p.Chips = 0

	var res *ActionResult
	if g.Phase != PhaseLobby && g.Phase != PhaseFinished && p.Status == StatusActive {
		// Auto-fold jika game sedang jalan
		r := g.HandleAction(playerName, Action{Type: ActionFold})
		res = &r
	}

	p.Status = StatusEliminated
	
	// Jika masih di Lobby, hapus player dari slice agar PlayerCount() akurat
	// dan slot pemain kosong kembali.
	if g.Phase == PhaseLobby {
		g.Players = append(g.Players[:pIndex], g.Players[pIndex+1:]...)
	}

	return remaining, true, res
}

// GetPlayerJID mengembalikan JID seorang player berdasarkan nama.
// Digunakan oleh PokerService untuk proper WhatsApp @-mentions.
func (g *Game) GetPlayerJID(name string) string {
	for _, p := range g.Players {
		if p.Name == name {
			return p.JID
		}
	}
	return ""
}

func (g *Game) getPlayer(name string) *Player {
	for _, p := range g.Players {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// activePlayers mengembalikan player yang belum tereliminasi.
func (g *Game) activePlayers() []*Player {
	var result []*Player
	for _, p := range g.Players {
		if p.Status != StatusEliminated {
			result = append(result, p)
		}
	}
	return result
}

// playersInHand mengembalikan player yang masih in hand (active atau all-in, bukan folded).
func (g *Game) playersInHand() []*Player {
	var result []*Player
	for _, p := range g.Players {
		if p.Status == StatusActive || p.Status == StatusAllIn {
			result = append(result, p)
		}
	}
	return result
}

// playersWhoCanAct mengembalikan player yang bisa bertindak (active, bukan all-in/folded).
func (g *Game) playersWhoCanAct() []*Player {
	var result []*Player
	for _, p := range g.Players {
		if p.Status == StatusActive && p.Chips > 0 {
			result = append(result, p)
		}
	}
	return result
}

// nextActivePlayerIndex mengembalikan index player aktif berikutnya setelah given index.
func (g *Game) nextActivePlayerIndex(fromIndex int) int {
	n := len(g.Players)
	for i := 1; i <= n; i++ {
		idx := (fromIndex + i) % n
		if g.Players[idx].Status != StatusEliminated {
			return idx
		}
	}
	return fromIndex // Seharusnya tidak terjadi
}

// nextPlayerWhoCanActFrom mengembalikan index player yang bisa bertindak
// berikutnya setelah given index.
func (g *Game) nextPlayerWhoCanActFrom(fromIndex int) int {
	n := len(g.Players)
	for i := 1; i <= n; i++ {
		idx := (fromIndex + i) % n
		if g.Players[idx].Status == StatusActive && g.Players[idx].Chips > 0 {
			return idx
		}
	}
	return fromIndex
}

// advanceToNextPlayer memajukan giliran ke player berikutnya yang bisa bertindak.
func (g *Game) advanceToNextPlayer() {
	n := len(g.Players)
	for i := 1; i <= n; i++ {
		idx := (g.currentTurnIndex + i) % n
		p := g.Players[idx]
		if p.Status == StatusActive && p.Chips > 0 {
			g.currentTurnIndex = idx
			return
		}
	}
}

// isBettingRoundComplete mengecek apakah semua player aktif sudah bertindak
// dan semua taruhan sama.
func (g *Game) isBettingRoundComplete() bool {
	canAct := g.playersWhoCanAct()
	if len(canAct) == 0 {
		return true
	}

	// Semua player yang bisa bertindak harus punya taruhan yang sama
	for _, p := range canAct {
		if p.CurrentBet != g.CurrentBetAmount {
			return false
		}
	}

	// Pastikan setiap player yang bisa bertindak sudah dapat giliran setidaknya sekali
	// Kecuali di pre-flop dimana big blind sudah memasang taruhan
	if g.actionsThisRound < len(canAct) {
		return false
	}

	return true
}

func (g *Game) totalPot() int {
	total := 0
	for _, p := range g.Players {
		total += p.TotalBet
	}
	return total
}

func (g *Game) eliminateBrokePlayers() {
	for _, p := range g.Players {
		if p.Status != StatusEliminated && p.Chips == 0 {
			p.Status = StatusEliminated
		}
	}
}

func (g *Game) isGameOver() bool {
	active := 0
	for _, p := range g.Players {
		if p.Status != StatusEliminated {
			active++
		}
	}
	return active <= 1
}
