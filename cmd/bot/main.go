package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"wa-gemini-bot/internal/ai"
	"wa-gemini-bot/internal/bot"
	"wa-gemini-bot/internal/config"
	"wa-gemini-bot/internal/memory"
	"wa-gemini-bot/internal/payment"
	"wa-gemini-bot/internal/trivia"
)

// main hanyalah "wiring" — menyambungkan modul-modul yang berdiri sendiri.
// Tidak ada business logic di sini, hanya urutan inisialisasi.
//
// Alur: Config → AI → Memory → [DOKU] → Bot → Start → Wait → Stop
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
		go doku.StartWebhookServer(cfg.DokuWebhookPort)
	} else {
		log.Println("DOKU tidak aktif — fitur donasi dinonaktifkan (set DOKU_* env vars untuk mengaktifkan)")
	}

	// Trivia — opsional, aktif hanya jika TRIVIA_ENABLED=true
	var triv *trivia.TriviaService
	if cfg.TriviaEnabled {
		triv = trivia.NewTriviaService(ai, cfg.AllowedGroupJIDs, cfg.TriviaIntervalMinutes, cfg.TriviaAnswerTimeoutSec)
		log.Printf("Trivia ready (interval: %d menit, timeout jawaban: %d detik)",
			cfg.TriviaIntervalMinutes, cfg.TriviaAnswerTimeoutSec)
	} else {
		log.Println("Trivia tidak aktif — set TRIVIA_ENABLED=true untuk mengaktifkan")
	}

	b, err := bot.NewBot(cfg, ai, mem, doku, triv)
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