package poker

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ==========================================================================
// PokerService — orchestrator yang mengelola game sessions per grup.
//
// Mengikuti pattern yang sama dengan TriviaService:
//   - Berkomunikasi dengan Bot via callback (tidak import package bot)
//   - Thread-safe via mutex
//   - Mengelola timer untuk timeout dan auto-advance
//
// Perbedaan dari Trivia: poker bersifat multi-turn dan interactive.
// Session bisa berlangsung beberapa menit dengan banyak aksi.
// ==========================================================================

// PokerService mengelola semua sesi poker aktif di semua grup.
type PokerService struct {
	mu       sync.Mutex
	sessions map[string]*pokerSession // groupJID → active session

	// Config
	startingChips    int
	smallBlind       int
	bigBlind         int
	turnTimeoutSec   int
	autoNextRoundSec int

	// Callbacks — diset oleh Bot via SetCallbacks.
	// Sama seperti TriviaService, ini adalah satu-satunya cara
	// PokerService berkomunikasi ke dunia luar.
	onSendGroupMessage      func(groupJID, text string)
	onSendGroupWithMentions func(groupJID, text string, mentionJIDs []string)
	onSendDM                func(userJID, text string)
	onRecordMemory          func(groupJID, sender, text string)
}

// pokerSession menyimpan state satu sesi poker di satu grup,
// termasuk game engine dan timer management.
type pokerSession struct {
	game         *Game
	lobbyTimer   *time.Timer // Countdown untuk lobby auto-start
	turnTimer    *time.Timer // Timeout giliran player
	roundTimer   *time.Timer // Delay sebelum ronde berikutnya
	creatorName  string      // Siapa yang memulai lobby
	roundNumber  int         // Nomor ronde saat ini
	startTime    time.Time   // Kapan game dimulai
	paused       bool        // true jika game sedang di-pause
	pausedBy     string      // Siapa yang pause
}

// NewPokerService membuat PokerService baru.
func NewPokerService(startingChips, smallBlind, bigBlind, turnTimeoutSec, autoNextRoundSec int) *PokerService {
	return &PokerService{
		sessions:         make(map[string]*pokerSession),
		startingChips:    startingChips,
		smallBlind:       smallBlind,
		bigBlind:         bigBlind,
		turnTimeoutSec:   turnTimeoutSec,
		autoNextRoundSec: autoNextRoundSec,
	}
}

// SetCallbacks mendaftarkan fungsi callback untuk berkomunikasi dengan Bot.
func (s *PokerService) SetCallbacks(
	sendGroupMsg func(groupJID, text string),
	sendGroupWithMentions func(groupJID, text string, mentionJIDs []string),
	sendDM func(userJID, text string),
	recordMem func(groupJID, sender, text string),
) {
	s.onSendGroupMessage = sendGroupMsg
	s.onSendGroupWithMentions = sendGroupWithMentions
	s.onSendDM = sendDM
	s.onRecordMemory = recordMem
}

// IsActive mengembalikan true jika ada sesi poker aktif di grup ini.
func (s *PokerService) IsActive(groupJID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[groupJID]
	return ok
}

// ==========================================================================
// Command handling — entry point dari Bot
// ==========================================================================

// pokerCommandRegex mencocokkan perintah poker saat di-mention.
// Contoh: "poker", "ikut", "mulai", "stop", "status", "pause", "lanjut"
var pokerCommandRegex = regexp.MustCompile(`(?i)^(poker|ikut|mulai|stop|status|pause|lanjut|resume)$`)

// actionRegex mencocokkan aksi poker (tanpa mention).
// Contoh: "fold", "check", "call", "raise 100", "bet 50", "allin"
var actionRegex = regexp.MustCompile(`(?i)^(fold|check|call|raise\s+\d+|bet\s+\d+|allin|all-in|all in)$`)

// HandleMentionCommand memproses perintah poker yang di-mention (e.g., "@Abdul poker").
// Return true jika pesan di-handle sebagai perintah poker.
func (s *PokerService) HandleMentionCommand(groupJID, senderName, senderJID, text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	if !pokerCommandRegex.MatchString(text) {
		return false
	}

	switch text {
	case "poker":
		s.handleNewLobby(groupJID, senderName, senderJID)
	case "ikut":
		s.handleJoin(groupJID, senderName, senderJID)
	case "mulai":
		s.handleStartGame(groupJID, senderName)
	case "stop":
		s.handleStop(groupJID, senderName)
	case "status":
		s.handleStatus(groupJID)
	case "pause":
		s.handlePause(groupJID, senderName)
	case "lanjut", "resume":
		s.handleResume(groupJID, senderName)
	default:
		return false
	}
	return true
}

// HandleGameAction memproses aksi poker (tanpa mention) saat game aktif.
// Return true jika pesan di-handle sebagai aksi poker.
func (s *PokerService) HandleGameAction(groupJID, senderName, text string) bool {
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

	// Hanya proses jika game sedang berjalan (bukan lobby)
	if session.game.Phase == PhaseLobby || session.game.Phase == PhaseFinished {
		s.mu.Unlock()
		return false
	}

	// Block aksi saat game di-pause
	if session.paused {
		s.mu.Unlock()
		s.sendGroup(groupJID, "⏸️ Game sedang di-pause. Ketik @Abdul lanjut untuk melanjutkan.")
		return true
	}

	// Parse aksi
	action := s.parseAction(text)
	if action == nil {
		s.mu.Unlock()
		return false
	}

	// Cancel turn timer sebelum proses aksi
	if session.turnTimer != nil {
		session.turnTimer.Stop()
	}

	result := session.game.HandleAction(senderName, *action)
	s.mu.Unlock()

	if !result.Valid {
		s.sendGroup(groupJID, result.Message)
		// Restart turn timer karena aksi invalid
		s.mu.Lock()
		if sess, ok := s.sessions[groupJID]; ok {
			s.startTurnTimer(groupJID, sess)
		}
		s.mu.Unlock()
		return true
	}

	// Kirim pesan aksi
	s.sendGroup(groupJID, result.Message)

	// Handle hasil aksi
	s.processActionResult(groupJID, result)
	return true
}

// ==========================================================================
// Command handlers
// ==========================================================================

func (s *PokerService) handleNewLobby(groupJID, senderName, senderJID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[groupJID]; ok {
		s.sendGroup(groupJID, "⚠️ Sudah ada game poker yang sedang berjalan di grup ini!")
		return
	}

	game := NewGame(s.smallBlind, s.bigBlind, s.startingChips)
	if err := game.AddPlayer(senderName, senderJID); err != nil {
		s.sendGroup(groupJID, "❌ Gagal membuat lobby: "+err.Error())
		return
	}

	session := &pokerSession{
		game:        game,
		creatorName: senderName,
	}

	// Set lobby timer — auto-start setelah 60 detik
	session.lobbyTimer = time.AfterFunc(60*time.Second, func() {
		s.handleLobbyTimeout(groupJID)
	})

	s.sessions[groupJID] = session

	msg := fmt.Sprintf(
		  "\n"+
			"   🃏 TEXAS HOLD'EM POKER 🃏      \n"+
			"\n"+
			"                                  \n"+
			"  Siapa mau main? Ketik:          \n"+
			"  👉 @Abdul ikut                   \n"+
			"                                  \n"+
			"  Min %d pemain, max %d pemain.     \n"+
			"  Chip awal: %s 💰             \n"+
			"  Blind: %d/%d                    \n"+
			"                                  \n"+
			"  Game dimulai dalam 60 detik     \n"+
			"  atau ketik: @Abdul mulai        \n"+
			"                                  \n"+
			"✅ %s bergabung! (1/%d)",
		MinPlayers, MaxPlayers,
		formatChips(s.startingChips),
		s.smallBlind, s.bigBlind,
		senderName, MaxPlayers,
	)
	s.sendGroup(groupJID, msg)
	s.recordMem(groupJID, "Abdul (Bot)", "[Poker lobby dibuat oleh "+senderName+"]")
}

func (s *PokerService) handleJoin(groupJID, senderName, senderJID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[groupJID]
	if !ok {
		s.sendGroup(groupJID, "❌ Belum ada lobby poker. Ketik @Abdul poker untuk memulai.")
		return
	}
	if session.game.Phase != PhaseLobby {
		s.sendGroup(groupJID, "⚠️ Game sudah berjalan — tunggu ronde berikutnya.")
		return
	}

	if err := session.game.AddPlayer(senderName, senderJID); err != nil {
		s.sendGroup(groupJID, "❌ "+err.Error())
		return
	}

	s.sendGroup(groupJID, fmt.Sprintf("✅ %s bergabung! (%d/%d)", senderName, session.game.PlayerCount(), MaxPlayers))
}

func (s *PokerService) handleStartGame(groupJID, senderName string) {
	s.mu.Lock()

	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Belum ada lobby poker.")
		return
	}
	if session.game.Phase != PhaseLobby {
		s.mu.Unlock()
		s.sendGroup(groupJID, "⚠️ Game sudah berjalan!")
		return
	}
	if session.game.PlayerCount() < MinPlayers {
		s.mu.Unlock()
		s.sendGroup(groupJID, fmt.Sprintf("❌ Butuh minimal %d pemain untuk mulai.", MinPlayers))
		return
	}

	// Stop lobby timer
	if session.lobbyTimer != nil {
		session.lobbyTimer.Stop()
	}

	session.startTime = time.Now()
	s.mu.Unlock()

	s.sendGroup(groupJID, fmt.Sprintf("🎮 Game dimulai dengan %d pemain!", session.game.PlayerCount()))
	s.startNewRound(groupJID)
}

func (s *PokerService) handleStop(groupJID, senderName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[groupJID]
	if !ok {
		s.sendGroup(groupJID, "❌ Tidak ada game poker yang berjalan.")
		return
	}

	// Cleanup timers
	s.cleanupSession(session)
	delete(s.sessions, groupJID)

	s.sendGroup(groupJID, fmt.Sprintf("🛑 Game poker dihentikan oleh %s.", senderName))
	s.recordMem(groupJID, "Abdul (Bot)", "[Poker dihentikan oleh "+senderName+"]")
}

func (s *PokerService) handlePause(groupJID, senderName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[groupJID]
	if !ok {
		s.sendGroup(groupJID, "❌ Tidak ada game poker yang berjalan.")
		return
	}
	if session.paused {
		s.sendGroup(groupJID, "⏸️ Game sudah di-pause.")
		return
	}

	session.paused = true
	session.pausedBy = senderName

	// Stop semua timer aktif — tidak ada timeout saat pause
	if session.turnTimer != nil {
		session.turnTimer.Stop()
	}
	if session.roundTimer != nil {
		session.roundTimer.Stop()
	}
	if session.lobbyTimer != nil {
		session.lobbyTimer.Stop()
	}

	s.sendGroup(groupJID, fmt.Sprintf("⏸️ Game di-pause oleh %s.\nKetik @Abdul lanjut untuk melanjutkan.", senderName))
}

func (s *PokerService) handleResume(groupJID, senderName string) {
	s.mu.Lock()

	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Tidak ada game poker yang berjalan.")
		return
	}
	if !session.paused {
		s.mu.Unlock()
		s.sendGroup(groupJID, "▶️ Game tidak sedang di-pause.")
		return
	}

	session.paused = false
	session.pausedBy = ""

	// Restart turn timer jika game sedang berjalan (bukan lobby)
	if session.game.Phase != PhaseLobby && session.game.Phase != PhaseFinished {
		s.startTurnTimer(groupJID, session)
		currentPlayer := session.game.GetCurrentTurnPlayer()
		s.mu.Unlock()

		s.sendGroup(groupJID, fmt.Sprintf("▶️ Game dilanjutkan oleh %s!", senderName))
		if currentPlayer != "" {
			s.sendTurnPrompt(groupJID, currentPlayer)
		}
	} else {
		s.mu.Unlock()
		s.sendGroup(groupJID, fmt.Sprintf("▶️ Game dilanjutkan oleh %s!", senderName))
	}
}

func (s *PokerService) handleStatus(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "ℹ️ Tidak ada game poker yang berjalan di grup ini.")
		return
	}

	game := session.game
	phase := game.Phase.String()
	currentTurn := game.GetCurrentTurnPlayer()
	pot := game.GetPot()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🃏 *POKER STATUS* — Ronde #%d\n", session.roundNumber))
	if session.paused {
		sb.WriteString("⏸️ *PAUSED*\n")
	}
	sb.WriteString(fmt.Sprintf("📍 Fase: %s\n", phase))
	sb.WriteString(fmt.Sprintf("💰 Pot: %s\n", formatChips(pot)))
	if currentTurn != "" {
		sb.WriteString(fmt.Sprintf("🎯 Giliran: %s\n", currentTurn))
	}

	if len(game.CommunityCards) > 0 {
		sb.WriteString(fmt.Sprintf("\n🃏 Kartu Meja:\n%s\n", RenderCards(game.CommunityCards)))
	}

	sb.WriteString("\n📊 Chip:\n")
	for _, standing := range game.GetChipStandings() {
		status := ""
		switch standing.Status {
		case StatusFolded:
			status = " (fold)"
		case StatusAllIn:
			status = " (all-in)"
		case StatusEliminated:
			status = " ☠️"
		}
		sb.WriteString(fmt.Sprintf("  • %s: %s 💰%s\n", standing.Name, formatChips(standing.Chips), status))
	}

	s.mu.Unlock()
	s.sendGroup(groupJID, sb.String())
}

// ==========================================================================
// Game flow
// ==========================================================================

func (s *PokerService) startNewRound(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	session.roundNumber++
	info, err := session.game.StartRound()
	if err != nil {
		s.cleanupSession(session)
		delete(s.sessions, groupJID)
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Gagal memulai ronde: "+err.Error())
		return
	}
	s.mu.Unlock()

	// Kirim info ronde ke grup
	var sb strings.Builder
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("🃏 RONDE #%d\n", session.roundNumber))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("🔘 Dealer: %s\n", info.DealerName))
	sb.WriteString(fmt.Sprintf("🔵 Small Blind: %s (%d 💰)\n", info.SmallBlindName, info.SmallBlindAmt))
	sb.WriteString(fmt.Sprintf("🔴 Big Blind: %s (%d 💰)\n", info.BigBlindName, info.BigBlindAmt))
	sb.WriteString(fmt.Sprintf("\n💰 Pot: %d\n", info.Pot))
	sb.WriteString("\n📍 Kartu kamu sudah dikirim via DM!\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	s.sendGroup(groupJID, sb.String())

	// Kirim hole cards via DM ke setiap player
	for _, p := range info.Players {
		cards := []Card{p.HoleCards[0], p.HoleCards[1]}
		dmMsg := fmt.Sprintf(
			"🃏 Kartu kamu (Ronde #%d):\n%s\n%s, %s",
			session.roundNumber,
			RenderCards(cards),
			p.HoleCards[0].FullName(), p.HoleCards[1].FullName(),
		)
		s.sendDM(p.JID, dmMsg)
	}

	// Kirim prompt giliran pertama
	s.sendTurnPrompt(groupJID, info.FirstTurnName)

	// Start turn timer
	s.mu.Lock()
	if sess, ok := s.sessions[groupJID]; ok {
		s.startTurnTimer(groupJID, sess)
	}
	s.mu.Unlock()
}

func (s *PokerService) sendTurnPrompt(groupJID, playerName string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	game := session.game
	currentBet := game.GetCurrentBet()
	playerBet := game.GetPlayerCurrentBet(playerName)
	playerChips := game.GetPlayerChips(playerName)
	pot := game.GetPot()
	playerJID := game.GetPlayerJID(playerName)
	s.mu.Unlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n🎯 Giliran: @%s\n", playerName))
	sb.WriteString(fmt.Sprintf("💰 Pot: %s | Taruhan saat ini: %d\n\n", formatChips(pot), currentBet))

	// Build daftar pilihan berdasarkan kondisi
	callAmount := currentBet - playerBet
	if callAmount > 0 {
		if callAmount > 0 && playerBet > 0 {
			sb.WriteString(fmt.Sprintf("(Sudah pasang %d, perlu tambah %d untuk call)\n", playerBet, callAmount))
		}
		sb.WriteString("Pilihan:\n")
		sb.WriteString(fmt.Sprintf("  • fold — menyerah\n"))
		sb.WriteString(fmt.Sprintf("  • call — samakan (%d 💰)\n", callAmount))
		sb.WriteString(fmt.Sprintf("  • raise [jumlah] — naikkan (min %d)\n", currentBet*2))
		sb.WriteString(fmt.Sprintf("  • allin — all in (%s 💰)\n", formatChips(playerChips)))
	} else {
		sb.WriteString("Pilihan:\n")
		sb.WriteString("  • check — lewat\n")
		minBet := game.BigBlind
		sb.WriteString(fmt.Sprintf("  • bet [jumlah] — taruhan (min %d)\n", minBet))
		sb.WriteString(fmt.Sprintf("  • allin — all in (%s 💰)\n", formatChips(playerChips)))
	}

	sb.WriteString(fmt.Sprintf("\n⏰ %d detik...", s.turnTimeoutSec))

	// Kirim dengan proper mention jika JID tersedia
	if playerJID != "" {
		s.sendGroupMention(groupJID, sb.String(), []string{playerJID})
	} else {
		s.sendGroup(groupJID, sb.String())
	}
}

func (s *PokerService) processActionResult(groupJID string, result ActionResult) {
	if result.RoundOver {
		s.handleRoundOver(groupJID, result)
		return
	}

	if result.PhaseEnded {
		s.sendPhaseMessage(groupJID, result)
	}

	if result.NextPlayer != "" {
		s.sendTurnPrompt(groupJID, result.NextPlayer)

		// Start turn timer
		s.mu.Lock()
		if sess, ok := s.sessions[groupJID]; ok {
			s.startTurnTimer(groupJID, sess)
		}
		s.mu.Unlock()
	}
}

func (s *PokerService) sendPhaseMessage(groupJID string, result ActionResult) {
	var sb strings.Builder

	switch result.NewPhase {
	case PhaseFlop:
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("🟢 THE FLOP\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	case PhaseTurn:
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("🟡 THE TURN\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	case PhaseRiver:
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		sb.WriteString("🔵 THE RIVER\n")
		sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	}

	sb.WriteString(RenderCards(result.CommunityCards))

	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	pot := 0
	if ok {
		pot = session.game.GetPot()
	}
	s.mu.Unlock()

	sb.WriteString(fmt.Sprintf("\n\n💰 Pot: %s\n", formatChips(pot)))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	s.sendGroup(groupJID, sb.String())
}

func (s *PokerService) handleRoundOver(groupJID string, result ActionResult) {
	// Cancel turn timer
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if ok && session.turnTimer != nil {
		session.turnTimer.Stop()
	}
	s.mu.Unlock()

	// Showdown results
	if len(result.ShowdownResults) > 0 {
		s.sendShowdownMessage(groupJID, result)
	}

	// Chip standings
	s.mu.Lock()
	session, ok = s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	standings := session.game.GetChipStandings()
	s.mu.Unlock()

	var sb strings.Builder
	sb.WriteString("\n📊 Sisa Chip:\n")
	for i, st := range standings {
		changeStr := ""
		if st.Change > 0 {
			changeStr = fmt.Sprintf(" (+%s)", formatChips(st.Change))
		} else if st.Change < 0 {
			changeStr = fmt.Sprintf(" (%s)", formatChips(st.Change))
		}
		status := ""
		if st.Status == StatusEliminated {
			status = " ☠️"
		}
		sb.WriteString(fmt.Sprintf("  %d. %s: %s 💰%s%s\n", i+1, st.Name, formatChips(st.Chips), changeStr, status))
	}

	s.sendGroup(groupJID, sb.String())

	// Game over?
	if result.GameOver {
		s.handleGameOver(groupJID, result, session)
		return
	}

	// Auto-start next round
	s.mu.Lock()
	session, ok = s.sessions[groupJID]
	if ok {
		session.game.PrepareNextRound()
		delay := time.Duration(s.autoNextRoundSec) * time.Second
		s.sendGroup(groupJID, fmt.Sprintf("\n🔄 Ronde berikutnya dalam %d detik...\n   Ketik @Abdul stop untuk berhenti.", s.autoNextRoundSec))
		session.roundTimer = time.AfterFunc(delay, func() {
			s.startNewRound(groupJID)
		})
	}
	s.mu.Unlock()
}

func (s *PokerService) sendShowdownMessage(groupJID string, result ActionResult) {
	var sb strings.Builder

	// Tampilkan community cards dulu
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("🏆 *SHOWDOWN!* 🏆\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(result.CommunityCards) > 0 {
		sb.WriteString("🃏 *Kartu Meja:*\n")
		sb.WriteString(RenderCards(result.CommunityCards))
		sb.WriteString("\n\n")
	}

	// Tampilkan kartu dan hand masing-masing player
	for _, sr := range result.ShowdownResults {
		cards := []Card{sr.HoleCards[0], sr.HoleCards[1]}
		sb.WriteString(fmt.Sprintf("👤 *%s*\n", sr.Name))
		sb.WriteString(fmt.Sprintf("   Kartu: %s\n", RenderCardsShort(cards)))
		sb.WriteString(fmt.Sprintf("   Tangan: *%s*\n", sr.BestHand.Description))
		// Tampilkan 5 kartu terbaik
		if len(sr.BestHand.BestCards) > 0 {
			sb.WriteString(fmt.Sprintf("   Kombinasi: %s\n", RenderCardsShort(sr.BestHand.BestCards)))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	for _, w := range result.Winners {
		sb.WriteString(fmt.Sprintf("🏆 *PEMENANG: %s*\n", w.PlayerName))
		sb.WriteString(fmt.Sprintf("💰 Menang: %s chip\n", formatChips(w.Amount)))
		if w.HandDesc != "" {
			sb.WriteString(fmt.Sprintf("✋ %s\n", w.HandDesc))
		}
	}
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	s.sendGroup(groupJID, sb.String())
}

func (s *PokerService) handleGameOver(groupJID string, result ActionResult, session *pokerSession) {
	duration := time.Since(session.startTime)
	minutes := int(duration.Minutes())

	var sb strings.Builder
	sb.WriteString("\n━━━━━━━ GAME OVER ━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("🏆 Pemenang: %s! 🎉\n", result.FinalWinner))
	sb.WriteString(fmt.Sprintf("Bermain %d ronde, total %d menit.\n", session.roundNumber, minutes))

	s.sendGroup(groupJID, sb.String())
	s.recordMem(groupJID, "Abdul (Bot)", fmt.Sprintf("[Poker selesai — pemenang: %s, %d ronde]", result.FinalWinner, session.roundNumber))

	// Cleanup session
	s.mu.Lock()
	s.cleanupSession(session)
	delete(s.sessions, groupJID)
	s.mu.Unlock()
}

// ==========================================================================
// Timer management
// ==========================================================================

func (s *PokerService) startTurnTimer(groupJID string, session *pokerSession) {
	// Cancel existing timer
	if session.turnTimer != nil {
		session.turnTimer.Stop()
	}

	timeout := time.Duration(s.turnTimeoutSec) * time.Second
	session.turnTimer = time.AfterFunc(timeout, func() {
		s.handleTurnTimeout(groupJID)
	})
}

func (s *PokerService) handleTurnTimeout(groupJID string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return
	}

	currentPlayer := session.game.GetCurrentTurnPlayer()
	if currentPlayer == "" {
		s.mu.Unlock()
		return
	}

	// Auto-fold
	result := session.game.HandleAction(currentPlayer, Action{Type: ActionFold})
	playerJID := session.game.GetPlayerJID(currentPlayer)
	s.mu.Unlock()

	// Mention player yang kena timeout
	msg := fmt.Sprintf("⏰ Waktu habis! @%s otomatis fold. 🏳️", currentPlayer)
	if playerJID != "" {
		s.sendGroupMention(groupJID, msg, []string{playerJID})
	} else {
		s.sendGroup(groupJID, msg)
	}

	if result.Valid {
		s.processActionResult(groupJID, result)
	}
}

func (s *PokerService) handleLobbyTimeout(groupJID string) {
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

	if session.game.PlayerCount() >= MinPlayers {
		session.startTime = time.Now()
		s.mu.Unlock()
		s.sendGroup(groupJID, fmt.Sprintf("⏰ Waktu lobby habis! Game dimulai dengan %d pemain!", session.game.PlayerCount()))
		s.startNewRound(groupJID)
	} else {
		s.cleanupSession(session)
		delete(s.sessions, groupJID)
		s.mu.Unlock()
		s.sendGroup(groupJID, "⏰ Waktu lobby habis — kurang pemain. Game dibatalkan.")
	}
}

// ==========================================================================
// Utilities
// ==========================================================================

func (s *PokerService) parseAction(text string) *Action {
	text = strings.TrimSpace(strings.ToLower(text))

	switch {
	case text == "fold":
		return &Action{Type: ActionFold}
	case text == "check":
		return &Action{Type: ActionCheck}
	case text == "call":
		return &Action{Type: ActionCall}
	case text == "allin" || text == "all-in" || text == "all in":
		return &Action{Type: ActionAllIn}
	case strings.HasPrefix(text, "raise"):
		parts := strings.Fields(text)
		if len(parts) != 2 {
			return nil
		}
		amount, err := strconv.Atoi(parts[1])
		if err != nil || amount <= 0 {
			return nil
		}
		return &Action{Type: ActionRaise, Amount: amount}
	case strings.HasPrefix(text, "bet"):
		parts := strings.Fields(text)
		if len(parts) != 2 {
			return nil
		}
		amount, err := strconv.Atoi(parts[1])
		if err != nil || amount <= 0 {
			return nil
		}
		return &Action{Type: ActionBet, Amount: amount}
	}
	return nil
}

func (s *PokerService) cleanupSession(session *pokerSession) {
	if session.lobbyTimer != nil {
		session.lobbyTimer.Stop()
	}
	if session.turnTimer != nil {
		session.turnTimer.Stop()
	}
	if session.roundTimer != nil {
		session.roundTimer.Stop()
	}
}

func (s *PokerService) sendGroup(groupJID, text string) {
	if s.onSendGroupMessage != nil {
		s.onSendGroupMessage(groupJID, text)
	}
}

// sendGroupMention mengirim pesan dengan proper WhatsApp @-mention.
// mentionJIDs berisi JID pemain yang ingin di-mention.
func (s *PokerService) sendGroupMention(groupJID, text string, mentionJIDs []string) {
	if s.onSendGroupWithMentions != nil {
		s.onSendGroupWithMentions(groupJID, text, mentionJIDs)
	} else {
		// Fallback ke plain text jika callback belum diset
		s.sendGroup(groupJID, text)
	}
}

func (s *PokerService) sendDM(userJID, text string) {
	if s.onSendDM != nil {
		s.onSendDM(userJID, text)
	} else {
		log.Printf("[POKER] ⚠️ sendDM callback not set, cannot send DM to %s", userJID)
	}
}

func (s *PokerService) recordMem(groupJID, sender, text string) {
	if s.onRecordMemory != nil {
		s.onRecordMemory(groupJID, sender, text)
	}
}

// formatChips memformat angka chip menjadi string yang readable.
// Contoh: 1000 → "1.000", -500 → "-500"
func formatChips(amount int) string {
	if amount == 0 {
		return "0"
	}

	negative := amount < 0
	if negative {
		amount = -amount
	}

	str := strconv.Itoa(amount)
	n := len(str)
	if n <= 3 {
		if negative {
			return "-" + str
		}
		return str
	}

	var result strings.Builder
	remainder := n % 3
	if remainder > 0 {
		result.WriteString(str[:remainder])
		result.WriteString(".")
	}
	for i := remainder; i < n; i += 3 {
		if i > remainder {
			result.WriteString(".")
		}
		result.WriteString(str[i : i+3])
	}

	if negative {
		return "-" + result.String()
	}
	return result.String()
}
