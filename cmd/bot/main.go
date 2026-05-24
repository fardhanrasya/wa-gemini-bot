package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"wa-gemini-bot/internal/ai"
	"wa-gemini-bot/internal/bot"
	"wa-gemini-bot/internal/config"
	"wa-gemini-bot/internal/economy"
	"wa-gemini-bot/internal/media"
	"wa-gemini-bot/internal/memory"
	"wa-gemini-bot/internal/payment"
	"wa-gemini-bot/internal/blackjack"
	"wa-gemini-bot/internal/poker"
	"wa-gemini-bot/internal/trivia"
)

// main hanyalah "wiring" — menyambungkan modul-modul yang berdiri sendiri.
// Tidak ada business logic di sini, hanya urutan inisialisasi.
//
// Alur: Config → AI → Memory → [DOKU] → [Trivia] → [Poker] → Bot → Start → Wait → Stop
//
// Setiap modul menerima dependency-nya via constructor (dependency injection).
// Ini membuat main.go menjadi satu-satunya tempat yang tahu tentang semua modul,
// sementara modul-modul itu sendiri tidak saling kenal — sesuai prinsip
// "reduce the number of places where each piece of knowledge is used."
func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Konfigurasi error: %v", err)
	}
	log.Printf("Bot aktif untuk %d grup: %v", len(cfg.AllowedGroupJIDs), cfg.AllowedGroupJIDs)

	ai, err := ai.NewAIService(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.SystemPrompt)
	if err != nil {
		log.Fatalf("AI service error: %v", err)
	}
	log.Printf("Gemini ready (model: %s, Google Search: ON)", cfg.GeminiModel)

	mem := memory.NewGroupMemory(cfg.MaxHistory)

	// DOKU — opsional, aktif hanya jika config terisi.
	// NewDokuService menerima *Config langsung (bukan struct perantara)
	// sehingga tidak ada information leakage antara Config dan DokuService.
	var doku *payment.DokuService
	if cfg.DokuEnabled {
		doku = payment.NewDokuService(cfg)
		log.Printf("DOKU Checkout ready (sandbox: %v, webhook port: %s)", cfg.DokuIsSandbox, cfg.DokuWebhookPort)
		// go doku.StartWebhookServer(cfg.DokuWebhookPort) // HTTP Server gabungan (Webhook + Admin Panel) dijalankan di dalam bot.Start()
	} else {
		log.Println("DOKU tidak aktif — fitur donasi dinonaktifkan (set DOKU_* env vars untuk mengaktifkan)")
	}

	// Trivia — opsional, aktif hanya jika TRIVIA_ENABLED=true
	var triv *trivia.TriviaService
	if cfg.TriviaEnabled {
		triv = trivia.NewTriviaService(ai, cfg.AllowedGroupJIDs, cfg.TriviaIntervalMinutes, cfg.TriviaAnswerTimeoutSec, cfg.TriviaReward)
		log.Printf("Trivia ready (interval: %d menit, timeout jawaban: %d detik, reward: %d chip)",
			cfg.TriviaIntervalMinutes, cfg.TriviaAnswerTimeoutSec, cfg.TriviaReward)
	} else {
		log.Println("Trivia tidak aktif — set TRIVIA_ENABLED=true untuk mengaktifkan")
	}

	// Poker — opsional, aktif hanya jika POKER_ENABLED=true
	var pok *poker.PokerService
	if cfg.PokerEnabled {
		pok = poker.NewPokerService(
			cfg.PokerStartingChips,
			cfg.PokerSmallBlind,
			cfg.PokerBigBlind,
			cfg.PokerTurnTimeoutSec,
			cfg.PokerAutoNextRoundSec,
		)
		log.Printf("Poker ready (chips: %d, blind: %d/%d, timeout: %ds, auto-next: %ds)",
			cfg.PokerStartingChips, cfg.PokerSmallBlind, cfg.PokerBigBlind,
			cfg.PokerTurnTimeoutSec, cfg.PokerAutoNextRoundSec)
	} else {
		log.Println("Poker tidak aktif — set POKER_ENABLED=true untuk mengaktifkan")
	}

	// Blackjack — opsional, aktif hanya jika BLACKJACK_ENABLED=true
	var bj *blackjack.BlackjackService
	if cfg.BlackjackEnabled {
		bj = blackjack.NewBlackjackService(
			cfg.BlackjackTurnTimeoutSec,
			cfg.BlackjackAutoNextRoundSec,
		)
		log.Printf("Blackjack ready (timeout: %ds, auto-next: %ds)",
			cfg.BlackjackTurnTimeoutSec, cfg.BlackjackAutoNextRoundSec)
	} else {
		log.Println("Blackjack tidak aktif — set BLACKJACK_ENABLED=true untuk mengaktifkan")
	}

	// Economy — selalu aktif sebagai base layer
	eco, err := economy.NewEconomyService("data/wa-economy.db")
	if err != nil {
		log.Fatalf("Economy service error: %v", err)
	}
	defer eco.Close()

	// Cloudinary — opsional, aktif jika CLOUDINARY_URL diset
	cld, err := media.NewCloudinaryService(cfg.CloudinaryURL)
	if err != nil {
		log.Fatalf("Cloudinary service error: %v", err)
	}
	if cld != nil {
		log.Println("Cloudinary ready (Image Upscaling: ON)")
	} else {
		log.Println("Cloudinary tidak aktif — set CLOUDINARY_URL untuk mengaktifkan fitur upscale")
	}

	b, err := bot.NewBot(cfg, ai, mem, doku, triv, pok, bj, eco, cld)
	if err != nil {
		log.Fatalf("Bot error: %v", err)
	}

	if err := b.Start(); err != nil {
		log.Fatalf("Gagal start bot: %v", err)
	}

	// Start trivia setelah bot connect agar callbacks sudah siap
	if triv != nil {
		triv.Start()
	}

	// Tunggu sampai dihentikan (Ctrl+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	b.Stop()
	log.Println("Bot dihentikan.")
}
