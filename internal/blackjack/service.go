package blackjack

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ==========================================================================
// BlackjackService — orchestrator yang mengelola game sessions per grup.
//
// Mengikuti arsitektur yang sama dengan PokerService:
//   - Berkomunikasi dengan Bot via callback
//   - Thread-safe via mutex
//   - Mengelola timer untuk lobby countdown, turn timeout, dan round cooldown
//   - Terintegrasi dengan EconomyService untuk transaksi chip via ledger
// ==========================================================================

const (
	DealerJID  = "dealer@abdul.bot" // JID khusus untuk dealer bot
	MinPlayers = 1
)

// formatPlayerBalance menampilkan total aset di meja (chip + bet aktif) dan bet ronde ini.
func formatPlayerBalance(p *BlackjackPlayer) string {
	total := p.Chips + p.Bet
	return fmt.Sprintf("Total di Meja: *%d* chip | Bet Aktif: *%d* chip", total, p.Bet)
}

// BlackjackService mengelola semua sesi blackjack aktif di grup.
type BlackjackService struct {
	mu       sync.Mutex
	sessions map[string]*blackjackSession // groupJID → active session

	// Config
	turnTimeoutSec   int
	autoNextRoundSec int

	// Callbacks — diset oleh Bot via SetCallbacks.
	onSendGroupMessage      func(groupJID, text string)
	onSendGroupWithMentions func(groupJID, text string, mentionJIDs []string)
	onSendDM                func(userJID, text string)
	onRecordMemory          func(groupJID, sender, text string)
	onAddBalance            func(jid string, amount int, txType, reference string) error
	onSubtractBalance       func(jid string, amount int, txType, reference string) error
}

type blackjackSession struct {
	game        *BlackjackGame
	lobbyTimer  *time.Timer // Countdown lobby
	turnTimer   *time.Timer // Timeout giliran player 60 detik (auto-stand)
	roundTimer  *time.Timer // Delay cooldown ronde berikutnya
	creatorName string      // Siapa yang memulai lobby
	roundNumber int         // Nomor ronde saat ini
	startTime   time.Time   // Kapan game dimulai
}

// NewBlackjackService membuat BlackjackService baru.
func NewBlackjackService(turnTimeoutSec, autoNextRoundSec int) *BlackjackService {
	return &BlackjackService{
		sessions:         make(map[string]*blackjackSession),
		turnTimeoutSec:   turnTimeoutSec,
		autoNextRoundSec: autoNextRoundSec,
	}
}

// SetCallbacks mendaftarkan fungsi callback untuk berkomunikasi dengan Bot.
func (s *BlackjackService) SetCallbacks(
	sendGroupMsg func(groupJID, text string),
	sendGroupWithMentions func(groupJID, text string, mentionJIDs []string),
	sendDM func(userJID, text string),
	recordMem func(groupJID, sender, text string),
	addBalance func(jid string, amount int, txType, reference string) error,
	subtractBalance func(jid string, amount int, txType, reference string) error,
) {
	s.onSendGroupMessage = sendGroupMsg
	s.onSendGroupWithMentions = sendGroupWithMentions
	s.onSendDM = sendDM
	s.onRecordMemory = recordMem
	s.onAddBalance = addBalance
	s.onSubtractBalance = subtractBalance
}

// IsActive mengembalikan true jika ada sesi blackjack aktif di grup ini.
func (s *BlackjackService) IsActive(groupJID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[groupJID]
	return ok
}

// IsPlayerInGame mengembalikan true jika pemain dengan senderJID sedang aktif di meja blackjack grup ini.
func (s *BlackjackService) IsPlayerInGame(groupJID, senderJID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[groupJID]
	if !ok {
		return false
	}
	return session.game.GetPlayerByJID(senderJID) != nil
}


// actionRegex mencocokkan aksi game blackjack (tanpa mention).
var actionRegex = regexp.MustCompile(`(?i)^(hit|stand|double)$`)

// HandleMentionCommand memproses perintah blackjack yang di-mention (e.g., "@bot bj").
func (s *BlackjackService) HandleMentionCommand(groupJID, senderName, senderJID, text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))

	// Normalisasi "blackjack" command ke "bj"
	if strings.HasPrefix(text, "blackjack") {
		text = "bj" + text[len("blackjack"):]
	}

	if !strings.HasPrefix(text, "bj") {
		return false
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		return false
	}

	// Pastikan command-nya adalah "bj"
	if parts[0] != "bj" {
		return false
	}

	// Jika hanya "@bot bj", buat lobby baru
	if len(parts) == 1 {
		s.handleNewLobby(groupJID, senderName, senderJID)
		return true
	}

	subCmd := parts[1]
	switch subCmd {
	case "help":
		s.handleHelp(groupJID)
	case "guide":
		s.handleGuide(groupJID)
	case "ikut":
		if len(parts) < 3 {
			s.sendGroup(groupJID, "❌ Format salah. Gunakan: @bot bj ikut <jumlah_taruhan> (contoh: @bot bj ikut 1000)")
			return true
		}
		amount, err := strconv.Atoi(parts[2])
		if err != nil || amount <= 0 {
			s.sendGroup(groupJID, "❌ Jumlah chip taruhan harus berupa angka positif.")
			return true
		}
		s.handleJoin(groupJID, senderName, senderJID, amount)
	case "mulai":
		s.handleStart(groupJID, senderName)
	case "leave":
		s.handleLeave(groupJID, senderName, senderJID)
	case "status":
		s.handleStatus(groupJID)
	case "bet", "taruhan":
		if len(parts) < 3 {
			s.sendGroup(groupJID, "❌ Format salah. Gunakan: @bot bj bet <jumlah> (contoh: @bot bj bet 500)")
			return true
		}
		amount, err := strconv.Atoi(parts[2])
		if err != nil || amount <= 0 {
			s.sendGroup(groupJID, "❌ Jumlah taruhan harus berupa angka positif.")
			return true
		}
		s.handleSetBet(groupJID, senderName, senderJID, amount)
	default:
		return false
	}

	return true
}

// HandleGameAction memproses aksi blackjack (tanpa mention) saat game aktif.
func (s *BlackjackService) HandleGameAction(groupJID, senderName, text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	if !actionRegex.MatchString(text) {
		return false
	}

	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return false
	}

	if session.game.Phase != PhasePlayerTurns {
		s.mu.Unlock()
		return false
	}

	// Hanya proses jika giliran aktif pemain ini
	if session.game.CurrentPlayer >= len(session.game.Players) {
		s.mu.Unlock()
		return false
	}

	currPlayer := session.game.Players[session.game.CurrentPlayer]
	if currPlayer.Name != senderName {
		s.mu.Unlock()
		return false
	}

	// Hentikan turn timer karena sudah merespon
	if session.turnTimer != nil {
		session.turnTimer.Stop()
	}
	s.mu.Unlock()

	switch text {
	case "hit":
		s.handleHit(groupJID, senderName)
	case "stand":
		s.handleStand(groupJID, senderName)
	case "double":
		s.handleDouble(groupJID, senderName)
	}

	return true
}

// handleNewLobby membuat lobby baru
func (s *BlackjackService) handleNewLobby(groupJID, senderName, senderJID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[groupJID]; ok {
		s.sendGroup(groupJID, "⚠️ Sudah ada game blackjack yang sedang berjalan di grup ini!")
		return
	}

	game := NewBlackjackGame()
	session := &blackjackSession{
		game:        game,
		creatorName: senderName,
		roundNumber: 0,
	}

	// Auto-start setelah 60 detik
	session.lobbyTimer = time.AfterFunc(60*time.Second, func() {
		s.handleLobbyTimeout(groupJID)
	})

	s.sessions[groupJID] = session

	msg := fmt.Sprintf(
		"🎰 *BLACKJACK (21) LOBBY* 🎰\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"Lobby dibuat oleh *%s*!\n"+
			"Siapa yang ingin ikut bermain? Ketik:\n"+
			"👉 *@bot bj ikut <jumlah_taruhan>*\n\n"+
			"• Maksimal 7 pemain per meja.\n"+
			"• Game akan dimulai otomatis dalam *60 detik*,\n"+
			"  atau ketik *@bot bj mulai* jika semua sudah siap.",
		senderName,
	)
	s.sendGroup(groupJID, msg)
	s.recordMem(groupJID, "bot", "[Blackjack lobby dibuat oleh "+senderName+"]")
}

func (s *BlackjackService) handleLobbyTimeout(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	if session.game.Phase != PhaseLobby {
		s.mu.Unlock()
		return
	}

	if session.game.PlayerCount() == 0 {
		delete(s.sessions, groupJID)
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Tidak ada pemain yang bergabung. Lobby blackjack dibatalkan.")
		return
	}
	s.mu.Unlock()

	s.startGame(groupJID)
}

func (s *BlackjackService) handleJoin(groupJID, senderName, senderJID string, bet int) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Belum ada lobby blackjack. Ketik @bot bj untuk memulai.")
		return
	}

	if session.game.Phase != PhaseLobby && session.game.Phase != PhaseFinished {
		s.mu.Unlock()
		s.sendGroup(groupJID, "⚠️ Game sedang berjalan — tunggu ronde berikutnya.")
		return
	}

	// 1. Cek apakah ini pemain lama yang ingin TOP-UP saldo meja
	existingPlayer := session.game.GetPlayerByJID(senderJID)
	if existingPlayer != nil {
		s.mu.Unlock()

		if s.onSubtractBalance != nil {
			if err := s.onSubtractBalance(senderJID, bet, "blackjack_bet", groupJID); err != nil {
				s.sendGroup(groupJID, fmt.Sprintf("❌ Gagal top-up: %v", err))
				return
			}
		}
		if s.onAddBalance != nil {
			if err := s.onAddBalance(DealerJID, bet, "blackjack_bet_dealer", groupJID); err != nil {
				if s.onAddBalance != nil {
					_ = s.onAddBalance(senderJID, bet, "blackjack_refund", groupJID)
				}
				s.sendGroup(groupJID, "❌ Sistem gagal mentransfer top-up ke dealer.")
				return
			}
		}

		s.mu.Lock()
		p := session.game.GetPlayerByJID(senderJID)
		if p != nil {
			p.Chips += bet
			// Sticky bet tidak diubah saat top-up — pemain atur via @bot bj bet <jumlah>
			s.sendGroup(groupJID, fmt.Sprintf("✅ *%s* melakukan top-up saldo meja sebesar *%d* chip! %s.", p.Name, bet, formatPlayerBalance(p)))
		}
		s.mu.Unlock()
		return
	}

	// 2. Ini pemain baru yang ingin bergabung (Buy-In)
	if session.game.PlayerCount() >= MaxPlayers {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Meja sudah penuh (maksimal 7 pemain).")
		return
	}
	wasEmptyTable := session.game.PlayerCount() == 0
	s.mu.Unlock()

	if s.onSubtractBalance != nil {
		if err := s.onSubtractBalance(senderJID, bet, "blackjack_bet", groupJID); err != nil {
			s.sendGroup(groupJID, fmt.Sprintf("❌ Gagal bergabung: %v", err))
			return
		}
	}

	if s.onAddBalance != nil {
		if err := s.onAddBalance(DealerJID, bet, "blackjack_bet_dealer", groupJID); err != nil {
			// Refund jika gagal
			if s.onAddBalance != nil {
				_ = s.onAddBalance(senderJID, bet, "blackjack_refund", groupJID)
			}
			s.sendGroup(groupJID, "❌ Sistem gagal mentransfer buy-in ke dealer.")
			return
		}
	}

	s.mu.Lock()
	session, ok = s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		if s.onAddBalance != nil {
			_ = s.onAddBalance(senderJID, bet, "blackjack_refund", groupJID)
		}
		if s.onSubtractBalance != nil {
			_ = s.onSubtractBalance(DealerJID, bet, "blackjack_refund_dealer", groupJID)
		}
		return
	}

	if err := session.game.AddPlayer(senderName, senderJID, bet); err != nil {
		s.mu.Unlock()
		// Refund jika error
		if s.onAddBalance != nil {
			_ = s.onAddBalance(senderJID, bet, "blackjack_refund", groupJID)
		}
		if s.onSubtractBalance != nil {
			_ = s.onSubtractBalance(DealerJID, bet, "blackjack_refund_dealer", groupJID)
		}
		s.sendGroup(groupJID, "❌ "+err.Error())
		return
	}

	playerCount := session.game.PlayerCount()
	p := session.game.GetPlayerByJID(senderJID)
	phase := session.game.Phase
	var balanceLine string
	if p != nil {
		balanceLine = formatPlayerBalance(p)
	}

	// Meja kosong (grace setelah bust): batalkan penutupan sesi, kembali ke lobby
	if wasEmptyTable {
		session.game.Phase = PhaseLobby
		if session.roundTimer != nil {
			session.roundTimer.Stop()
			session.roundTimer = nil
		}
		if session.lobbyTimer != nil {
			session.lobbyTimer.Stop()
		}
		session.lobbyTimer = time.AfterFunc(60*time.Second, func() {
			s.handleLobbyTimeout(groupJID)
		})
		phase = PhaseLobby
	}
	s.mu.Unlock()

	joinMsg := fmt.Sprintf("✅ *%s* bergabung! %s (%d/%d)", senderName, balanceLine, playerCount, MaxPlayers)
	if phase == PhaseFinished {
		joinMsg += fmt.Sprintf("\n⏰ Ronde berikutnya dimulai dalam *%d detik*... Ubah taruhan dengan *@bot bj bet <jumlah>*.", s.autoNextRoundSec)
	} else {
		joinMsg += "\nKetik *@bot bj mulai* atau tunggu lobby otomatis."
	}
	s.sendGroup(groupJID, joinMsg)
}

func (s *BlackjackService) handleSetBet(groupJID, senderName, senderJID string, bet int) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Tidak ada game blackjack yang sedang aktif.")
		return
	}

	err := session.game.SetPlayerBet(senderName, bet)
	s.mu.Unlock()

	if err != nil {
		s.sendGroup(groupJID, "❌ "+err.Error())
		return
	}

	s.sendGroup(groupJID, fmt.Sprintf("✅ Taruhan *@%s* diubah menjadi *%d* chip untuk ronde berikutnya.", senderName, bet))
}

func (s *BlackjackService) handleStart(groupJID, senderName string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Belum ada lobby blackjack.")
		return
	}

	if session.game.Phase != PhaseLobby {
		s.mu.Unlock()
		s.sendGroup(groupJID, "⚠️ Game sudah berjalan!")
		return
	}

	if session.game.PlayerCount() == 0 {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Butuh minimal 1 pemain untuk mulai.")
		return
	}

	if session.lobbyTimer != nil {
		session.lobbyTimer.Stop()
	}
	s.mu.Unlock()

	s.startGame(groupJID)
}

func (s *BlackjackService) startGame(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	session.roundNumber++
	session.startTime = time.Now()

	roundInfo, err := session.game.StartRound()
	if err != nil {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Gagal memulai game: "+err.Error())
		return
	}

	roundNum := session.roundNumber
	phase := session.game.Phase
	dealerUp := session.game.DealerUpCard
	currentPlayerIdx := session.game.CurrentPlayer

	type playerSnapshot struct {
		name, jid       string
		cards           []Card
		handValue       int
		handType        string
		bet, chips      int
		status          PlayerStatus
	}
	players := make([]playerSnapshot, len(session.game.Players))
	for i, p := range session.game.Players {
		players[i] = playerSnapshot{
			name:      p.Name,
			jid:       p.JID,
			cards:     append([]Card(nil), p.Hand.Cards...),
			handValue: p.Hand.Value(),
			handType:  s.getHandTypeString(p.Hand),
			bet:       p.Bet,
			chips:     p.Chips,
			status:    p.Status,
		}
	}
	s.mu.Unlock()

	// 1. Kirim DM ke pemain
	for _, p := range players {
		dmText := fmt.Sprintf(
			"🃏 *KARTU KAMU (Blackjack Ronde #%d)* 🃏\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
				"%s\n"+
				"Total nilai: *%d* (%s)\n"+
				"Total di Meja: *%d* chip | Bet Aktif: *%d* chip\n"+
				"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
				"Ketik langsung *hit*, *stand*, atau *double* di grup untuk bermain!",
			roundNum,
			RenderCards(p.cards),
			p.handValue,
			p.handType,
			p.chips+p.bet,
			p.bet,
		)
		s.sendDM(p.jid, dmText)
	}

	// 2. Tampilkan info awal di grup
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎮 *BLACKJACK — RONDE #%d* 🎮\n", roundNum))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString(fmt.Sprintf("🎰 *Dealer Upcard*: %s\n\n", RenderCards([]Card{dealerUp})))
	if roundInfo != nil && len(roundInfo.BetAdjustments) > 0 {
		sb.WriteString("⚠️ *Penyesuaian taruhan otomatis (all-in):*\n")
		for _, adj := range roundInfo.BetAdjustments {
			sb.WriteString(fmt.Sprintf("  • *%s*: Bet Aktif diturunkan *%d* → *%d* chip (all-in, total meja tidak cukup)\n",
				adj.PlayerName, adj.OldBet, adj.NewBet))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("👥 *Status Pemain*:\n")
	for _, p := range players {
		statusStr := ""
		if p.status == StatusBlackjack {
			statusStr = " 🔥 *BLACKJACK!*"
		}
		sb.WriteString(fmt.Sprintf("  • *%s*: %s%s\n", p.name, formatPlayerBalance(&BlackjackPlayer{Chips: p.chips, Bet: p.bet}), statusStr))
	}
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if phase == PhaseFinished {
		sb.WriteString("⚠️ *Dealer memiliki Blackjack natural!* Game berakhir.\n\n")

		s.mu.Lock()
		session, ok := s.sessions[groupJID]
		if !ok {
			s.mu.Unlock()
			return
		}
		results := session.game.DetermineWinners()
		s.applyTablePayouts(groupJID, session.game, results)
		s.mu.Unlock()

		sb.WriteString(s.formatResultsBlock(results))
		sb.WriteString(s.formatCooldownFooter())

		s.sendGroup(groupJID, sb.String())
		s.scheduleRoundCooldown(groupJID)
		return
	}

	// Cari pemain aktif pertama
	activePlayer := ""
	activeJID := ""
	if currentPlayerIdx < len(players) {
		activePlayer = players[currentPlayerIdx].name
		activeJID = players[currentPlayerIdx].jid
	}

	if activePlayer != "" {
		sb.WriteString(fmt.Sprintf("🎯 Giliran: *@%s*\n", activePlayer))
		sb.WriteString("Ketik langsung: *hit*, *stand*, atau *double*")

		s.sendGroupMention(groupJID, sb.String(), []string{activeJID})

		s.mu.Lock()
		if session, ok := s.sessions[groupJID]; ok {
			s.startTurnTimer(groupJID, session)
		}
		s.mu.Unlock()
	} else {
		s.sendGroup(groupJID, sb.String())
		s.playDealerTurn(groupJID)
	}
}

func (s *BlackjackService) getHandTypeString(h *Hand) string {
	if h.IsSoft() {
		return "Soft"
	}
	return "Hard"
}

func (s *BlackjackService) handleHit(groupJID, senderName string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	res, err := session.game.Hit(senderName)
	if err != nil {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ "+err.Error())
		return
	}
	
	player := session.game.GetPlayer(senderName)
	newCard := player.Hand.Cards[len(player.Hand.Cards)-1]
	playerJID := player.JID
	playerHandValue := player.Hand.Value()
	playerHandCards := player.Hand.Cards
	s.mu.Unlock()

	// Kirim kartu ke DM
	dmText := fmt.Sprintf(
		"🃏 *KARTU BARU (Blackjack)* 🃏\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
			"Anda menarik: %s\n"+
			"Hand saat ini:\n"+
			"%s\n"+
			"Total nilai: *%d* (%s)\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
		newCard.String(),
		RenderCards(playerHandCards),
		playerHandValue,
		s.getHandTypeString(player.Hand),
	)
	s.sendDM(playerJID, dmText)

	// Update grup
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🃏 *%s* melakukan *Hit* dan menerima kartu baru.\n", senderName))
	
	if res.PlayerBust {
		sb.WriteString(fmt.Sprintf("💥 *Bust!* Total nilai kartu %s adalah *%d* (melebihi 21).\n", senderName, playerHandValue))
	} else if playerHandValue == 21 {
		sb.WriteString(fmt.Sprintf("✨ *%s* memiliki total nilai *21*!\n", senderName))
	} else {
		sb.WriteString(fmt.Sprintf("Total kartu %s saat ini: *%d*.\n", senderName, playerHandValue))
	}

	s.processActionResult(groupJID, res, sb.String())
}

func (s *BlackjackService) handleStand(groupJID, senderName string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	res, err := session.game.Stand(senderName)
	if err != nil {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ "+err.Error())
		return
	}

	playerVal := session.game.GetPlayer(senderName).Hand.Value()
	s.mu.Unlock()

	msg := fmt.Sprintf("➡️ *%s* memilih *Stand* dengan nilai kartu *%d*.\n", senderName, playerVal)
	s.processActionResult(groupJID, res, msg)
}

func (s *BlackjackService) handleDouble(groupJID, senderName string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	player := session.game.GetPlayer(senderName)
	if player == nil {
		s.mu.Unlock()
		return
	}
	originalBet := player.Bet
	playerJID := player.JID

	res, err := session.game.DoubleDown(senderName)
	if err != nil {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ "+err.Error())
		s.mu.Lock()
		if session, ok = s.sessions[groupJID]; ok {
			s.startTurnTimer(groupJID, session)
		}
		s.mu.Unlock()
		return
	}

	newCard := player.Hand.Cards[len(player.Hand.Cards)-1]
	playerHandValue := player.Hand.Value()
	playerHandCards := player.Hand.Cards
	s.mu.Unlock()

	// DM kartu baru
	dmText := fmt.Sprintf(
		"🃏 *DOUBLE DOWN CARD* 🃏\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
			"Anda Double Down dan menarik: %s\n"+
			"Hand final:\n"+
			"%s\n"+
			"Total nilai: *%d* (%s)\n"+
			"Taruhan total: *%d* chip (Ganda!)\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
		newCard.String(),
		RenderCards(playerHandCards),
		playerHandValue,
		s.getHandTypeString(player.Hand),
		originalBet*2,
	)
	s.sendDM(playerJID, dmText)

	// Update grup
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("💰 *%s* melakukan *Double Down*! Taruhan dilipatgandakan menjadi *%d* chip.\n", senderName, originalBet*2))
	if res.PlayerBust {
		sb.WriteString(fmt.Sprintf("💥 *Bust!* Kartu ketiga menyebabkan total melebihi 21 (*%d*).\n", playerHandValue))
	} else {
		sb.WriteString(fmt.Sprintf("Pemain menerima kartu tambahan dan otomatis *Stand* dengan nilai *%d*.\n", playerHandValue))
	}

	s.processActionResult(groupJID, res, sb.String())
}

func (s *BlackjackService) processActionResult(groupJID string, res *ActionResult, msg string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	var sb strings.Builder
	if msg != "" {
		sb.WriteString(msg)
		sb.WriteString("\n")
	}

	if res.DealerTurn {
		s.mu.Unlock()
		s.sendGroup(groupJID, sb.String())
		s.playDealerTurn(groupJID)
		return
	}

	if res.NextPlayer != "" {
		sb.WriteString(fmt.Sprintf("🎯 Giliran berikutnya: *@%s*\n", res.NextPlayer))
		sb.WriteString("Ketik langsung: *hit*, *stand*, atau *double*")
		
		playerJID := session.game.GetPlayer(res.NextPlayer).JID
		s.mu.Unlock()

		s.sendGroupMention(groupJID, sb.String(), []string{playerJID})

		s.mu.Lock()
		s.startTurnTimer(groupJID, session)
		s.mu.Unlock()
	} else {
		s.mu.Unlock()
		s.sendGroup(groupJID, sb.String())
	}
}

func (s *BlackjackService) playDealerTurn(groupJID string) {
	s.mu.Lock()
	_, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	s.sendGroup(groupJID, "🎰 *Giliran Dealer!* Membuka kartu tertutup...")
	time.Sleep(1500 * time.Millisecond)

	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	res := session.game.PlayDealerTurn()
	results := session.game.DetermineWinners()
	s.mu.Unlock()

	var sb strings.Builder
	sb.WriteString("🎰 *DEALER TURN RESULT* 🎰\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString(fmt.Sprintf("🃏 Kartu Dealer: %s\n", RenderCards(res.Cards)))
	sb.WriteString(fmt.Sprintf("Total nilai: *%d*\n\n", res.FinalValue))

	if res.IsBust {
		sb.WriteString("💥 *Dealer BUST!* Semua pemain yang tidak bust otomatis menang.\n")
	} else {
		sb.WriteString(fmt.Sprintf("Dealer bertahan dengan nilai *%d*.\n", res.FinalValue))
	}
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	sb.WriteString("🏆 *BLACKJACK RESULTS* 🏆\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	s.mu.Lock()
	s.applyTablePayouts(groupJID, session.game, results)
	s.mu.Unlock()

	sb.WriteString(s.formatResultsBlock(results))
	sb.WriteString(s.formatCooldownFooter())

	s.sendGroup(groupJID, sb.String())
	s.scheduleRoundCooldown(groupJID)
}

func (s *BlackjackService) handleRoundCooldownTimeout(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	// 1. Identifikasi pemain yang kehabisan chip di meja (setelah ronde selesai)
	type playerRemoval struct {
		name string
		jid  string
	}
	var playersToRemove []playerRemoval
	for _, p := range session.game.Players {
		if p.Chips <= 0 {
			playersToRemove = append(playersToRemove, playerRemoval{name: p.Name, jid: p.JID})
		}
	}

	// 2. Hapus pemain yang kehabisan chip dari meja (tidak ada refund — chip meja sudah 0)
	for _, pr := range playersToRemove {
		session.game.RemovePlayer(pr.name)
	}
	s.mu.Unlock()

	// Kirim notifikasi jika ada pemain yang dikeluarkan
	for _, pr := range playersToRemove {
		s.sendGroup(groupJID, fmt.Sprintf("⚠️ *%s* dikeluarkan dari meja blackjack karena kehabisan chip di meja (*0* chip). Silakan buy-in lagi dengan *@bot bj ikut <nominal>*.", pr.name))
	}

	s.mu.Lock()
	session, ok = s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	// Reset game state (clear hands)
	session.game.Reset()

	// Jika sudah tidak ada pemain tersisa di meja, beri jeda grace period 14 detik
	if session.game.PlayerCount() == 0 {
		session.game.Phase = PhaseFinished
		if session.roundTimer != nil {
			session.roundTimer.Stop()
		}
		session.roundTimer = time.AfterFunc(14*time.Second, func() {
			s.handleGraceTimeout(groupJID)
		})
		s.mu.Unlock()

		s.sendGroup(groupJID, "🏁 Tidak ada pemain tersisa di meja.\n⏰ Menunggu pemain baru bergabung dalam *14 detik* sebelum game ditutup...")
		return
	}
	s.mu.Unlock()

	s.startGame(groupJID)
}

func (s *BlackjackService) handleGraceTimeout(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	if session.game.PlayerCount() == 0 {
		s.cleanupSession(session)
		delete(s.sessions, groupJID)
		s.mu.Unlock()
		s.sendGroup(groupJID, "🏁 Game blackjack telah selesai karena tidak ada pemain tersisa di meja.")
		return
	}
	s.mu.Unlock()
}

func (s *BlackjackService) handleLeave(groupJID, senderName, senderJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Tidak ada game blackjack yang sedang aktif.")
		return
	}

	player := session.game.GetPlayerByJID(senderJID)
	if player == nil {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Kamu tidak sedang berada di meja game ini.")
		return
	}

	isPlayerTurn := false
	if session.game.Phase == PhasePlayerTurns && session.game.CurrentPlayer < len(session.game.Players) {
		isPlayerTurn = (session.game.Players[session.game.CurrentPlayer].JID == senderJID)
	}

	if isPlayerTurn && session.turnTimer != nil {
		session.turnTimer.Stop()
	}

	// Refund sisa saldo meja pemain ke saldo utama (sama seperti poker)
	chipsToRefund := player.Chips
	s.mu.Unlock()

	refunded := false
	if chipsToRefund > 0 {
		if s.onAddBalance != nil {
			s.refundTableChips(senderJID, chipsToRefund, groupJID)
			refunded = true
		}
	}

	s.mu.Lock()
	session, ok = s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	session.game.RemovePlayer(senderName)

	if session.game.PlayerCount() == 0 {
		session.game.Phase = PhaseFinished
		if session.roundTimer != nil {
			session.roundTimer.Stop()
		}
		session.roundTimer = time.AfterFunc(14*time.Second, func() {
			s.handleGraceTimeout(groupJID)
		})
		s.mu.Unlock()

		if refunded {
			s.sendGroup(groupJID, fmt.Sprintf("👋 *%s* meninggalkan meja. Sisa saldo meja *%d* chip telah dikembalikan ke saldo utama.\n⏰ Meja kosong! Menunggu pemain baru bergabung dalam *14 detik* sebelum game ditutup...", senderName, chipsToRefund))
		} else {
			s.sendGroup(groupJID, fmt.Sprintf("👋 *%s* meninggalkan meja. Tidak ada pemain tersisa.\n⏰ Meja kosong! Menunggu pemain baru bergabung dalam *14 detik* sebelum game ditutup...", senderName))
		}
		return
	}

	s.mu.Unlock()
	if refunded {
		s.sendGroup(groupJID, fmt.Sprintf("👋 *%s* meninggalkan meja game. Sisa saldo meja *%d* chip telah dikembalikan ke saldo utama.", senderName, chipsToRefund))
	} else {
		s.sendGroup(groupJID, fmt.Sprintf("👋 *%s* meninggalkan meja game.", senderName))
	}

	if isPlayerTurn {
		res := &ActionResult{
			Valid: true,
		}
		s.mu.Lock()
		session.game.advanceTurn(res)
		s.mu.Unlock()
		s.processActionResult(groupJID, res, "")
	}
}

func (s *BlackjackService) startTurnTimer(groupJID string, session *blackjackSession) {
	if session.turnTimer != nil {
		session.turnTimer.Stop()
	}

	session.turnTimer = time.AfterFunc(time.Duration(s.turnTimeoutSec)*time.Second, func() {
		s.handleTurnTimeout(groupJID)
	})
}

func (s *BlackjackService) handleTurnTimeout(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	if session.game.Phase != PhasePlayerTurns || session.game.CurrentPlayer >= len(session.game.Players) {
		s.mu.Unlock()
		return
	}

	currPlayer := session.game.Players[session.game.CurrentPlayer]
	playerName := currPlayer.Name
	s.mu.Unlock()

	s.sendGroup(groupJID, fmt.Sprintf("⏰ *Timeout!* *@%s* terlalu lama merespon, otomatis *Stand*.", playerName))
	s.handleStand(groupJID, playerName)
}


func (s *BlackjackService) handleStatus(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "ℹ️ Tidak ada game blackjack yang sedang berjalan di grup ini.")
		return
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎰 *STATUS BLACKJACK (Ronde #%d)* 🎰\n", session.roundNumber))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString(fmt.Sprintf("📍 Fase: *%s*\n", session.game.Phase.String()))
	
	if session.game.Phase != PhaseLobby {
		sb.WriteString(fmt.Sprintf("🃏 Dealer Upcard: %s\n\n", RenderCards([]Card{session.game.DealerUpCard})))
	}

	sb.WriteString("👥 *Status Pemain*:\n")
	for i, p := range session.game.Players {
		statusDetail := ""
		if session.game.Phase == PhasePlayerTurns {
			if i == session.game.CurrentPlayer {
				statusDetail = " 👈 *Giliran aktif*"
			} else {
				statusDetail = fmt.Sprintf(" (%s)", p.Status.String())
			}
		} else {
			statusDetail = fmt.Sprintf(" (%s)", p.Status.String())
		}
		sb.WriteString(fmt.Sprintf("  • *%s*: %s%s\n", p.Name, formatPlayerBalance(p), statusDetail))
	}
	sb.WriteString("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	s.mu.Unlock()
	s.sendGroup(groupJID, sb.String())
}

func (s *BlackjackService) handleGuide(groupJID string) {
	msg := "🎰 *BLACKJACK (21) — PANDUAN BERMAIN* 🎰\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
		"Tujuan utama Blackjack adalah mengalahkan kartu Dealer tanpa melebihi total nilai *21*.\n\n" +
		"*1. Nilai Kartu:*\n" +
		"• *Kartu Angka (2-10)*: Sesuai angka tertulis.\n" +
		"• *Face Cards (J, Q, K)*: Bernilai *10*.\n" +
		"• *Ace (As)*: Sangat fleksibel! Bisa bernilai *1* atau *11* (otomatis dihitung yang paling menguntungkan).\n\n" +
		"*2. Pilihan Aksi (Ketik langsung di grup):*\n" +
		"👉 *hit* — Menambah 1 kartu baru.\n" +
		"👉 *stand* — Bertahan dengan nilai kartu saat ini.\n" +
		"👉 *double* — Menggandakan taruhan awal, menerima tepat *1* kartu tambahan, lalu otomatis *stand*.\n\n" +
		"*3. Kondisi & Payout (Rasio Bayaran):*\n" +
		"• *Blackjack Natural* (Ace + 10-value dari 2 kartu awal): Dibayar *3:2* (2.5x taruhan).\n" +
		"• *Kemenangan Biasa*: Dibayar *1:1* (2x taruhan).\n" +
		"• *Seri (Push)*: Kartu bernilai sama dengan dealer, taruhan dikembalikan (1x taruhan).\n" +
		"• *Kalah* / *Bust* (>21): Taruhan ditarik bandar.\n\n" +
		"*4. Taruhan:*\n" +
		"• Taruhan ronde berikutnya mengikuti ronde sebelumnya (bukan all-in otomatis).\n" +
		"• Ubah taruhan saat jeda antar ronde: `@bot bj bet <jumlah>`.\n" +
		"• Jika saldo meja kurang dari taruhan, sistem otomatis all-in (taruhan = seluruh saldo meja).\n\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"Mulai meja game baru sekarang dengan mengetik *@bot bj*!"
	s.sendGroup(groupJID, msg)
}

func (s *BlackjackService) handleHelp(groupJID string) {
	msg := "🎰 *BLACKJACK COMMANDS — DAFTAR PERINTAH* 🎰\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
		"Gunakan perintah berikut dengan me-mention bot:\n\n" +
		"*Mengelola Game (Pake Tag/Mention):*\n" +
		"👉 `@bot bj` — Membuat lobby game blackjack baru.\n" +
		"👉 `@bot bj ikut <saldo>` — Beli/top-up saldo meja blackjack (contoh: `@bot bj ikut 5000`).\n" +
		"👉 `@bot bj bet <jumlah>` — Ubah taruhan antar ronde saat jeda cooldown (contoh: `@bot bj bet 10000`).\n" +
		"👉 `@bot bj mulai` — Memulai pembagian kartu secara manual.\n" +
		"👉 `@bot bj status` — Melihat status permainan dan giliran saat ini.\n" +
		"👉 `@bot bj leave` — Meninggalkan meja (sisa saldo meja dikembalikan ke wallet).\n" +
		"👉 `@bot bj guide` — Melihat panduan cara bermain blackjack.\n" +
		"👉 `@bot bj help` — Melihat daftar perintah blackjack ini.\n\n" +
		"*Aksi di Meja Game (Ketik Langsung Tanpa Tag Bot):*\n" +
		"Ketik langsung saat giliran aktif Anda:\n" +
		"👉 `hit` — Ambil kartu tambahan.\n" +
		"👉 `stand` — Selesai mengambil kartu.\n" +
		"👉 `double` — Gandakan taruhan dan ambil 1 kartu terakhir.\n\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	s.sendGroup(groupJID, msg)
}

// applyTablePayouts menambahkan payout ke saldo meja dan memotong bank dealer.
func (s *BlackjackService) applyTablePayouts(groupJID string, game *BlackjackGame, results []WinResult) {
	for _, r := range results {
		p := game.GetPlayer(r.PlayerName)
		if p != nil {
			p.Chips += r.Payout
		}
		if r.Payout > 0 && s.onSubtractBalance != nil {
			_ = s.onSubtractBalance(DealerJID, r.Payout, "blackjack_payout", groupJID)
		}
	}
}

func (s *BlackjackService) formatResultsBlock(results []WinResult) string {
	var sb strings.Builder
	for _, r := range results {
		var detail string
		switch r.Outcome {
		case "blackjack":
			detail = fmt.Sprintf("🔥 *BLACKJACK!* Menang *%d* chip (3:2 payout)", r.Payout)
		case "win":
			detail = fmt.Sprintf("✅ *MENANG* biasa! Menerima *%d* chip (1:1 payout)", r.Payout)
		case "push":
			detail = fmt.Sprintf("🤝 *SERI (Push)*. Taruhan *%d* chip dikembalikan", r.Payout)
		case "lose":
			detail = "❌ *KALAH*. Taruhan disita bandar"
		}
		sb.WriteString(fmt.Sprintf("👤 *%s*: %s\n", r.PlayerName, detail))
	}
	return sb.String()
}

func (s *BlackjackService) formatCooldownFooter() string {
	return fmt.Sprintf(
		"\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
			"⏰ Ronde berikutnya dimulai dalam *%d detik*...\n"+
			"Ubah taruhan dengan *@bot bj bet <jumlah>*.\n"+
			"Pemain baru dapat mengetik *@bot bj ikut <taruhan>* untuk bergabung.",
		s.autoNextRoundSec,
	)
}

func (s *BlackjackService) scheduleRoundCooldown(groupJID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[groupJID]
	if !ok {
		return
	}
	if session.roundTimer != nil {
		session.roundTimer.Stop()
	}
	delay := time.Duration(s.autoNextRoundSec) * time.Second
	session.roundTimer = time.AfterFunc(delay, func() {
		s.handleRoundCooldownTimeout(groupJID)
	})
}

func (s *BlackjackService) refundTableChips(jid string, chips int, groupJID string) {
	if chips <= 0 {
		return
	}
	if s.onSubtractBalance != nil {
		_ = s.onSubtractBalance(DealerJID, chips, "blackjack_refund_dealer", groupJID)
	}
	if s.onAddBalance != nil {
		_ = s.onAddBalance(jid, chips, "blackjack_refund", groupJID)
	}
}

func (s *BlackjackService) cleanupSession(session *blackjackSession) {
	if session.lobbyTimer != nil {
		session.lobbyTimer.Stop()
	}
	if session.turnTimer != nil {
		session.turnTimer.Stop()
	}
	if session.roundTimer != nil {
		session.roundTimer.Stop()
	}

	// Refund semua sisa chip pemain yang masih di meja ke saldo utama
	for _, p := range session.game.Players {
		s.refundTableChips(p.JID, p.Chips, "")
	}
}

// Low-level helper methods to abstract bot communications
func (s *BlackjackService) sendGroup(groupJID, text string) {
	if s.onSendGroupMessage != nil {
		s.onSendGroupMessage(groupJID, text)
	}
}

func (s *BlackjackService) sendGroupMention(groupJID, text string, mentionJIDs []string) {
	if s.onSendGroupWithMentions != nil {
		s.onSendGroupWithMentions(groupJID, text, mentionJIDs)
	}
}

func (s *BlackjackService) sendDM(userJID, text string) {
	if s.onSendDM != nil {
		s.onSendDM(userJID, text)
	}
}

func (s *BlackjackService) recordMem(groupJID, sender, text string) {
	if s.onRecordMemory != nil {
		s.onRecordMemory(groupJID, sender, text)
	}
}
