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
	onAddBalance            func(jid string, amount int) error
	onSubtractBalance       func(jid string, amount int) error
}

// pokerSession menyimpan state satu sesi poker di satu grup,
// termasuk game engine dan timer management.
type pokerSession struct {
	game        *Game
	lobbyTimer  *time.Timer // Countdown untuk lobby auto-start
	turnTimer   *time.Timer // Timeout giliran player
	roundTimer  *time.Timer // Delay sebelum ronde berikutnya
	creatorName string      // Siapa yang memulai lobby
	roundNumber int         // Nomor ronde saat ini
	startTime   time.Time   // Kapan game dimulai
	paused      bool        // true jika game sedang di-pause
	pausedBy    string      // Siapa yang pause
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
	addBalance func(jid string, amount int) error,
	subtractBalance func(jid string, amount int) error,
) {
	s.onSendGroupMessage = sendGroupMsg
	s.onSendGroupWithMentions = sendGroupWithMentions
	s.onSendDM = sendDM
	s.onRecordMemory = recordMem
	s.onAddBalance = addBalance
	s.onSubtractBalance = subtractBalance
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
// Contoh: "poker", "poker help", "poker guide", "ikut 1000", "mulai", "stop", "status", "pause", "lanjut", "resume", "keluar", "leave"
var pokerCommandRegex = regexp.MustCompile(`(?i)^(poker(?:\s+(?:help|guide))?|ikut(?:\s+\d+)?|mulai|stop|status|pause|lanjut|resume|keluar|leave)$`)

// actionRegex mencocokkan aksi poker (tanpa mention).
// Contoh: "fold", "check", "call", "raise 100", "bet 50", "allin"
var actionRegex = regexp.MustCompile(`(?i)^(fold|check|call|raise\s+\d+|bet\s+\d+|allin|all-in|all in)$`)

// HandleMentionCommand memproses perintah poker yang di-mention (e.g., "@bot poker").
// Return true jika pesan di-handle sebagai perintah poker.
func (s *PokerService) HandleMentionCommand(groupJID, senderName, senderJID, text string) bool {
	text = strings.TrimSpace(strings.ToLower(text))
	if !pokerCommandRegex.MatchString(text) {
		return false
	}

	// Menggunakan strings.Fields untuk mengabaikan spasi ganda secara otomatis (define errors out of existence).
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return false
	}
	cmd := parts[0]

	switch cmd {
	case "poker":
		if len(parts) > 1 {
			subCmd := parts[1]
			switch subCmd {
			case "help":
				s.handlePokerHelp(groupJID)
				return true
			case "guide":
				s.handlePokerGuide(groupJID)
				return true
			}
		}
		s.handleNewLobby(groupJID, senderName, senderJID)
	case "ikut":
		if len(parts) < 2 {
			s.sendGroup(groupJID, "❌ Format salah. Gunakan: @bot ikut <jumlah_chip> (contoh: @bot ikut 1000)")
			return true
		}
		amount, err := strconv.Atoi(parts[1])
		if err != nil || amount <= 0 {
			s.sendGroup(groupJID, "❌ Jumlah chip harus angka positif.")
			return true
		}
		s.handleJoin(groupJID, senderName, senderJID, amount)
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
	case "keluar", "leave":
		s.handleLeave(groupJID, senderName)
	default:
		return false
	}
	return true
}

// handlePokerHelp mengirimkan petunjuk penggunaan perintah poker.
func (s *PokerService) handlePokerHelp(groupJID string) {
	msg := "🃏 *POKER HELP — PANDUAN PERINTAH BOT* 🃏\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
		"Untuk berinteraksi dengan bot poker, gunakan perintah-perintah berikut dengan melakukan `@mention` pada bot:\n\n" +
		"*Lobby & Pengaturan Game:*\n" +
		"👉 `@bot poker`\n" +
		"   Membuat lobby game poker baru di grup.\n" +
		"👉 `@bot ikut <jumlah>`\n" +
		"   Bergabung ke lobby dengan jumlah chip tertentu (misal: `@bot ikut 1000`).\n" +
		"👉 `@bot mulai`\n" +
		"   Memulai permainan (minimal 2 pemain).\n" +
		"👉 `@bot status`\n" +
		"   Melihat status permainan saat ini, kartu meja, dan sisa chip pemain.\n" +
		"👉 `@bot stop`\n" +
		"   Menghentikan permainan secara paksa dan mengembalikan seluruh chip ke saldo.\n" +
		"👉 `@bot keluar` / `@bot leave`\n" +
		"   Keluar dari permainan dan menarik sisa chip kembali ke saldo.\n" +
		"👉 `@bot pause`\n" +
		"   Menunda permainan sementara (timer giliran dihentikan).\n" +
		"👉 `@bot lanjut` / `@bot resume`\n" +
		"   Melanjutkan kembali permainan yang sedang di-pause.\n" +
		"👉 `@bot poker guide`\n" +
		"   Melihat panduan lengkap cara bermain Texas Hold'em Poker.\n\n" +
		"*Aksi di Meja Game (Tanpa Tag Bot):*\n" +
		"Saat giliranmu, ketik langsung perintah berikut tanpa me-mention bot:\n" +
		"👉 `fold` — Menyerah dan membuang kartu.\n" +
		"👉 `check` — Melewati giliran tanpa menambah taruhan (jika tidak ada taruhan).\n" +
		"👉 `call` — Menyamakan taruhan saat ini.\n" +
		"👉 `bet <jumlah>` — Memasang taruhan baru (misal: `bet 100`).\n" +
		"👉 `raise <jumlah>` — Menaikkan nilai taruhan (misal: `raise 200`).\n" +
		"👉 `allin` — Mempertaruhkan seluruh chip yang tersisa di meja.\n\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"Ketik `@bot poker guide` untuk membaca panduan aturan & kombinasi kartu poker!"

	s.sendGroup(groupJID, msg)
}

// handlePokerGuide mengirimkan panduan cara bermain Texas Hold'em Poker.
func (s *PokerService) handlePokerGuide(groupJID string) {
	msg := "🃏 *TEXAS HOLD'EM POKER — CARA BERMAIN* 🃏\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n" +
		"Texas Hold'em adalah permainan kartu di mana tujuanmu adalah memenangkan chip di dalam *pot* dengan membuat kombinasi 5 kartu terbaik atau membuat pemain lain menyerah (*fold*).\n\n" +
		"*1. Jalannya Permainan (Fase)*\n" +
		"• *Pre-Flop*: Setiap pemain mendapat 2 kartu rahasia (*hole cards*) via DM. Taruhan blind dipasang secara otomatis oleh Dealer.\n" +
		"• *Flop*: 3 kartu meja (*community cards*) dibuka untuk semua pemain. Ronde taruhan dimulai.\n" +
		"• *Turn*: Kartu meja ke-4 dibuka. Ronde taruhan dimulai lagi.\n" +
		"• *River*: Kartu meja ke-5 (terakhir) dibuka. Ronde taruhan terakhir.\n" +
		"• *Showdown*: Pemain yang tersisa membuka kartunya, kombinasi 5 kartu terbaik memenangkan seluruh chip di pot!\n\n" +
		"*2. Urutan Kekuatan Kombinasi Kartu (Terendah → Tertinggi)*\n" +
		"1. *High Card* 🃏 — Kartu dengan nilai tertinggi jika tidak ada kombinasi.\n" +
		"2. *One Pair* 👥 — 2 kartu dengan nilai yang sama (contoh: A-A).\n" +
		"3. *Two Pair* ✌️ — 2 pasang kartu dengan nilai yang sama (contoh: K-K & 10-10).\n" +
		"4. *Three of a Kind* 🌲 — 3 kartu dengan nilai yang sama (contoh: 8-8-8).\n" +
		"5. *Straight* 🪜 — 5 kartu berurutan nilainya, beda lambang (contoh: 5-6-7-8-9).\n" +
		"6. *Flush* 💧 — 5 kartu dengan lambang yang sama, acak nilainya (contoh: semua Sekop/Spade ♠️).\n" +
		"7. *Full House* 🏠 — Gabungan *Three of a Kind* dan *One Pair* (contoh: Q-Q-Q & 7-7).\n" +
		"8. *Four of a Kind* 🍀 — 4 kartu dengan nilai yang sama (contoh: J-J-J-J).\n" +
		"9. *Straight Flush* 🌪️ — 5 kartu berurutan nilainya DAN lambangnya sama (contoh: 4-5-6-7-8 Hati/Heart ♥️).\n" +
		"10. *Royal Flush* 👑 — 5 kartu berurutan tertinggi dengan lambang sama (10-J-Q-K-A Sekop/Spade ♠️).\n\n" +
		"*3. Tips & Trik*\n" +
		"• Jangan takut untuk *fold* jika kartu awalmu buruk.\n" +
		"• Perhatikan kartu meja dan potensi kombinasi kartu lawan.\n" +
		"• Kelola chipmu dengan bijak, pasang taruhan lebih besar saat kartumu kuat untuk memaksimalkan pot!\n\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"Ayo buat meja sekarang dengan mengetik `@bot poker`!"

	s.sendGroup(groupJID, msg)
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
		s.sendGroup(groupJID, "⏸️ Game sedang di-pause. Ketik @bot lanjut untuk melanjutkan.")
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

	// Handle hasil aksi (message akan digabung dan dikirim di sini)
	s.processActionResult(groupJID, result, nil)
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
			"  Lobby dibuat oleh %s!           \n"+
			"  Siapa mau main? Ketik:          \n"+
			"  👉 @bot ikut <jumlah>          \n"+
			"                                  \n"+
			"  Min %d pemain, max %d pemain.     \n"+
			"  Blind: %d/%d                    \n"+
			"                                  \n"+
			"  Game dimulai dalam 60 detik     \n"+
			"  atau ketik: @bot mulai        \n",
		senderName,
		MinPlayers, MaxPlayers,
		s.smallBlind, s.bigBlind,
	)
	s.sendGroup(groupJID, msg)
	s.recordMem(groupJID, "bot (Bot)", "[Poker lobby dibuat oleh "+senderName+"]")
}

func (s *PokerService) handleJoin(groupJID, senderName, senderJID string, buyin int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[groupJID]
	if !ok {
		s.sendGroup(groupJID, "❌ Belum ada lobby poker. Ketik @bot poker untuk memulai.")
		return
	}
	if session.game.Phase != PhaseLobby {
		s.sendGroup(groupJID, "⚠️ Game sudah berjalan — tunggu ronde berikutnya.")
		return
	}

	// Potong saldo dari DB (gagal jika saldo tak cukup)
	if s.onSubtractBalance != nil {
		if err := s.onSubtractBalance(senderJID, buyin); err != nil {
			s.sendGroup(groupJID, fmt.Sprintf("❌ Gagal join: %v", err))
			return
		}
	} else {
		// Fallback jika callback belum diset
		log.Printf("[POKER] Warning: onSubtractBalance nil, using fake chips")
	}

	// Update game.AddPlayer untuk menerima buyin. Kita harus update game.AddPlayer di game.go!
	// Sementara kita akan set p.Chips secara manual atau update metode AddPlayer nanti.
	if err := session.game.AddPlayer(senderName, senderJID); err != nil {
		// Refund jika gagal join
		if s.onAddBalance != nil {
			_ = s.onAddBalance(senderJID, buyin)
		}
		s.sendGroup(groupJID, "❌ "+err.Error())
		return
	}

	// Set chip awal player sejumlah buy-in
	p := session.game.getPlayer(senderName)
	if p != nil {
		p.Chips = buyin
		p.BuyIn = buyin // Track buy-in amount untuk perhitungan change
	}

	s.sendGroup(groupJID, fmt.Sprintf("✅ %s bergabung bawa %s chip! (%d/%d)", senderName, formatChips(buyin), session.game.PlayerCount(), MaxPlayers))
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

	// Refund semua chip yang ada di meja
	refunds := session.game.RefundAll()
	if s.onAddBalance != nil {
		for name, amount := range refunds {
			jid := session.game.GetPlayerJID(name)
			if jid != "" {
				_ = s.onAddBalance(jid, amount)
			}
		}
	}

	// Cleanup timers
	s.cleanupSession(session)
	delete(s.sessions, groupJID)

	s.sendGroup(groupJID, fmt.Sprintf("🛑 Game poker dihentikan oleh %s. Seluruh chip di meja telah dikembalikan ke saldo masing-masing.", senderName))
	s.recordMem(groupJID, "bot (Bot)", "[Poker dihentikan oleh "+senderName+"]")
}

func (s *PokerService) handleLeave(groupJID, senderName string) {
	s.mu.Lock()

	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Tidak ada game poker yang berjalan.")
		return
	}

	// Proses leave di core game (auto-fold jika perlu, ambil sisa chip)
	remaining, found, result := session.game.Leave(senderName)
	if !found {
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Kamu tidak sedang bermain di meja ini.")
		return
	}

	// Cancel turn timer lama
	if session.turnTimer != nil {
		session.turnTimer.Stop()
	}
	s.mu.Unlock()

	// Refund sisa chip ke DB
	jid := session.game.GetPlayerJID(senderName)
	if s.onAddBalance != nil && jid != "" && remaining > 0 {
		_ = s.onAddBalance(jid, remaining)
	}

	s.sendGroup(groupJID, fmt.Sprintf("👋 %s keluar dari meja dan membawa pulang %s chip.", senderName, formatChips(remaining)))

	// Lanjutkan ke player berikutnya jika game masih berjalan dan giliran berubah
	if result != nil {
		s.processActionResult(groupJID, *result, nil)
	}
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

	s.sendGroup(groupJID, fmt.Sprintf("⏸️ Game di-pause oleh %s.\nKetik @bot lanjut untuk melanjutkan.", senderName))
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
			prompt, mentions := s.buildTurnPrompt(groupJID, currentPlayer)
			if len(mentions) > 0 {
				s.sendGroupMention(groupJID, prompt, mentions)
			} else {
				s.sendGroup(groupJID, prompt)
			}
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
		// Refund chip sebagai fallback jika start round gagal
		refunds := session.game.RefundAll()
		if s.onAddBalance != nil {
			for name, amount := range refunds {
				jid := session.game.GetPlayerJID(name)
				if jid != "" {
					_ = s.onAddBalance(jid, amount)
				}
			}
		}

		s.cleanupSession(session)
		delete(s.sessions, groupJID)
		s.mu.Unlock()
		s.sendGroup(groupJID, "❌ Gagal memulai ronde: "+err.Error()+" (Seluruh chip dikembalikan)")
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
	prompt, mentions := s.buildTurnPrompt(groupJID, info.FirstTurnName)
	if len(mentions) > 0 {
		s.sendGroupMention(groupJID, prompt, mentions)
	} else {
		s.sendGroup(groupJID, prompt)
	}

	// Start turn timer
	s.mu.Lock()
	if sess, ok := s.sessions[groupJID]; ok {
		s.startTurnTimer(groupJID, sess)
	}
	s.mu.Unlock()
}

func (s *PokerService) buildTurnPrompt(groupJID, playerName string) (string, []string) {
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if !ok {
		s.mu.Unlock()
		return "", nil
	}

	game := session.game
	currentBet := game.GetCurrentBet()
	playerBet := game.GetPlayerCurrentBet(playerName)
	playerChips := game.GetPlayerChips(playerName)
	pot := game.GetPot()
	playerJID := game.GetPlayerJID(playerName)
	s.mu.Unlock()

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎯 Giliran: @%s\n", playerName))
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

	var mentions []string
	if playerJID != "" {
		mentions = append(mentions, playerJID)
	}

	return sb.String(), mentions
}

func (s *PokerService) processActionResult(groupJID string, result ActionResult, prefixMentions []string) {
	var finalMsg strings.Builder
	
	if result.Message != "" {
		finalMsg.WriteString(result.Message)
		finalMsg.WriteString("\n\n")
	}

	if result.RoundOver {
		s.handleRoundOver(groupJID, result, &finalMsg, prefixMentions)
		return
	}

	if result.PhaseEnded {
		finalMsg.WriteString(s.buildPhaseMessage(groupJID, result))
		finalMsg.WriteString("\n\n")
	}

	if result.NextPlayer != "" {
		prompt, mentions := s.buildTurnPrompt(groupJID, result.NextPlayer)
		finalMsg.WriteString(prompt)
		prefixMentions = append(prefixMentions, mentions...)
		
		s.sendGroupMention(groupJID, strings.TrimSpace(finalMsg.String()), prefixMentions)

		// Start turn timer
		s.mu.Lock()
		if sess, ok := s.sessions[groupJID]; ok {
			s.startTurnTimer(groupJID, sess)
		}
		s.mu.Unlock()
	} else {
		if finalMsg.Len() > 0 {
			if len(prefixMentions) > 0 {
				s.sendGroupMention(groupJID, strings.TrimSpace(finalMsg.String()), prefixMentions)
			} else {
				s.sendGroup(groupJID, strings.TrimSpace(finalMsg.String()))
			}
		}
	}
}

func (s *PokerService) buildPhaseMessage(groupJID string, result ActionResult) string {
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

	return sb.String()
}

func (s *PokerService) handleRoundOver(groupJID string, result ActionResult, finalMsg *strings.Builder, mentions []string) {
	// Cancel turn timer
	s.mu.Lock()
	session, ok := s.sessions[groupJID]
	if ok && session.turnTimer != nil {
		session.turnTimer.Stop()
	}
	s.mu.Unlock()

	// Showdown results
	if len(result.ShowdownResults) > 0 {
		finalMsg.WriteString(s.buildShowdownMessage(result))
		finalMsg.WriteString("\n")
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

	finalMsg.WriteString("\n📊 Sisa Chip:\n")
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
		finalMsg.WriteString(fmt.Sprintf("  %d. %s: %s 💰%s%s\n", i+1, st.Name, formatChips(st.Chips), changeStr, status))
	}

	// Game over?
	if result.GameOver {
		if len(mentions) > 0 {
			s.sendGroupMention(groupJID, strings.TrimSpace(finalMsg.String()), mentions)
		} else {
			s.sendGroup(groupJID, strings.TrimSpace(finalMsg.String()))
		}
		s.handleGameOver(groupJID, result, session)
		return
	}

	// Auto-start next round
	s.mu.Lock()
	session, ok = s.sessions[groupJID]
	if ok {
		session.game.PrepareNextRound()
		delay := time.Duration(s.autoNextRoundSec) * time.Second
		finalMsg.WriteString(fmt.Sprintf("\n\n🔄 Ronde berikutnya dalam %d detik...\n   Ketik @bot stop untuk berhenti.", s.autoNextRoundSec))
		session.roundTimer = time.AfterFunc(delay, func() {
			s.startNewRound(groupJID)
		})
	}
	s.mu.Unlock()

	if len(mentions) > 0 {
		s.sendGroupMention(groupJID, strings.TrimSpace(finalMsg.String()), mentions)
	} else {
		s.sendGroup(groupJID, strings.TrimSpace(finalMsg.String()))
	}
}

func (s *PokerService) buildShowdownMessage(result ActionResult) string {
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
	
	// Accumulate winnings for players who won multiple pots (e.g., main pot and side pots)
	type accumulatedWin struct {
		Amount   int
		HandDesc string
	}
	accWinners := make(map[string]*accumulatedWin)
	var orderedNames []string // To preserve the order of winners
	
	for _, w := range result.Winners {
		if acc, exists := accWinners[w.PlayerName]; exists {
			acc.Amount += w.Amount
		} else {
			accWinners[w.PlayerName] = &accumulatedWin{
				Amount:   w.Amount,
				HandDesc: w.HandDesc,
			}
			orderedNames = append(orderedNames, w.PlayerName)
		}
	}

	for _, name := range orderedNames {
		w := accWinners[name]
		sb.WriteString(fmt.Sprintf("🏆 *PEMENANG: %s*\n", name))
		sb.WriteString(fmt.Sprintf("💰 Menang: %s chip\n", formatChips(w.Amount)))
		if w.HandDesc != "" {
			sb.WriteString(fmt.Sprintf("✋ %s\n", w.HandDesc))
		}
	}
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return sb.String()
}

func (s *PokerService) handleGameOver(groupJID string, result ActionResult, session *pokerSession) {
	duration := time.Since(session.startTime)
	minutes := int(duration.Minutes())

	var sb strings.Builder
	sb.WriteString("\n━━━━━━━ GAME OVER ━━━━━━━\n")
	sb.WriteString(fmt.Sprintf("🏆 Pemenang: %s! 🎉\n", result.FinalWinner))
	sb.WriteString(fmt.Sprintf("Bermain %d ronde, total %d menit.\n", session.roundNumber, minutes))

	s.sendGroup(groupJID, sb.String())
	s.recordMem(groupJID, "bot (Bot)", fmt.Sprintf("[Poker selesai — pemenang: %s, %d ronde]", result.FinalWinner, session.roundNumber))

	// Refund sisa chip ke DB (biasanya pemenang akhir membawa semua chip)
	s.mu.Lock()
	refunds := session.game.RefundAll()
	if s.onAddBalance != nil {
		for name, amount := range refunds {
			jid := session.game.GetPlayerJID(name)
			if jid != "" {
				_ = s.onAddBalance(jid, amount)
			}
		}
	}

	// Cleanup session
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

	// Override result.Message dengan message timeout
	result.Message = fmt.Sprintf("⏰ Waktu habis! @%s otomatis fold. 🏳️", currentPlayer)
	var mentions []string
	if playerJID != "" {
		mentions = append(mentions, playerJID)
	}

	if result.Valid {
		s.processActionResult(groupJID, result, mentions)
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
		// Refund chip karena game batal
		refunds := session.game.RefundAll()
		if s.onAddBalance != nil {
			for name, amount := range refunds {
				jid := session.game.GetPlayerJID(name)
				if jid != "" {
					_ = s.onAddBalance(jid, amount)
				}
			}
		}

		s.cleanupSession(session)
		delete(s.sessions, groupJID)
		s.mu.Unlock()
		s.sendGroup(groupJID, "⏰ Waktu lobby habis — kurang pemain. Game dibatalkan dan seluruh chip dikembalikan.")
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
