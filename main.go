package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// main hanyalah "wiring" — menyambungkan module-module yang sudah didesain
// agar masing-masing bisa berdiri sendiri. Tidak ada business logic di sini.
//
// Alur: Config → AI Service → Memory → Bot → Start → Wait → Stop
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

	bot, err := NewBot(cfg, ai, memory)
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