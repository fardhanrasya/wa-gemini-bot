package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
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
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Konfigurasi error: %v", err)
	}
	log.Printf("Bot aktif untuk %d grup: %v", len(cfg.AllowedGroupJIDs), cfg.AllowedGroupJIDs)

	ai, err := NewAIService(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.SystemPrompt)
	if err != nil {
		log.Fatalf("AI service error: %v", err)
	}
	log.Printf("Gemini ready (model: %s, Google Search: ON)", cfg.GeminiModel)

	memory := NewGroupMemory(cfg.MaxHistory)

	// DOKU — opsional, aktif hanya jika config terisi.
	// NewDokuService menerima *Config langsung (bukan struct perantara)
	// sehingga tidak ada information leakage antara Config dan DokuService.
	var doku *DokuService
	if cfg.DokuEnabled {
		doku = NewDokuService(cfg)
		log.Printf("DOKU Checkout ready (sandbox: %v, webhook port: %s)", cfg.DokuIsSandbox, cfg.DokuWebhookPort)
		go doku.StartWebhookServer(cfg.DokuWebhookPort)
	} else {
		log.Println("DOKU tidak aktif — fitur donasi dinonaktifkan (set DOKU_* env vars untuk mengaktifkan)")
	}

	bot, err := NewBot(cfg, ai, memory, doku)
	if err != nil {
		log.Fatalf("Bot error: %v", err)
	}

	if err := bot.Start(); err != nil {
		log.Fatalf("Gagal start bot: %v", err)
	}

	// Tunggu sampai dihentikan (Ctrl+C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	bot.Stop()
	log.Println("Bot dihentikan.")
}