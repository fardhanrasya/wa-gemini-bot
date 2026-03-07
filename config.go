package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// Config menyimpan semua konfigurasi aplikasi.
// Semua validasi dan default value ditangani di sini sehingga
// kode lain tidak perlu tahu dari mana konfigurasi berasal.
type Config struct {
	GeminiAPIKey    string
	GeminiModel     string
	SystemPrompt    string
	AllowedGroupJIDs []string
	MaxHistory      int
}

// LoadConfig membaca konfigurasi dari .env dan environment variables.
// Mengembalikan error yang jelas jika ada konfigurasi wajib yang missing,
// sehingga caller tidak perlu menangani partial-config state.
func LoadConfig() (*Config, error) {
	// Ignore error — .env is optional if env vars are set directly
	_ = godotenv.Load()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY belum di-set (cek file .env)")
	}

	allowedStr := os.Getenv("ALLOWED_GROUP_JID")
	if allowedStr == "" {
		return nil, fmt.Errorf("ALLOWED_GROUP_JID belum di-set (cek file .env)")
	}

	var allowedJIDs []string
	for _, jid := range strings.Split(allowedStr, ",") {
		jid = strings.TrimSpace(jid)
		if jid != "" {
			allowedJIDs = append(allowedJIDs, jid)
		}
	}
	if len(allowedJIDs) == 0 {
		return nil, fmt.Errorf("ALLOWED_GROUP_JID kosong setelah di-parse")
	}

	// Defaults yang masuk akal — caller tidak perlu khawatir soal ini
	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}

	systemPrompt := os.Getenv("SYSTEM_PROMPT")
	if systemPrompt == "" {
		systemPrompt = `Kamu adalah Abdul, asisten AI cerdas di grup WhatsApp. Kamu memiliki kemampuan berikut:
1. Memahami konteks obrolan grup dari riwayat percakapan yang diberikan.
2. Menjawab pertanyaan dengan informatif dan akurat.
3. Jika ditanya tentang berita terbaru, topik trending, atau informasi terkini, GUNAKAN Google Search untuk mendapatkan informasi paling update.
4. Gunakan bahasa yang santai dan ramah, sesuai dengan gaya chat grup WhatsApp Indonesia.
5. Jika ada riwayat obrolan, rujuk konteks pembicaraan sebelumnya kalau relevan.
6. Jawab secara ringkas tapi informatif, jangan terlalu panjang kecuali diminta detail.`
	}

	maxHistory := 20 // Bisa di-override via env kalau perlu nanti

	return &Config{
		GeminiAPIKey:    apiKey,
		GeminiModel:     model,
		SystemPrompt:    systemPrompt,
		AllowedGroupJIDs: allowedJIDs,
		MaxHistory:      maxHistory,
	}, nil
}

// IsAllowedGroup memeriksa apakah sebuah group JID ada di daftar yang diizinkan.
// Method ini ada di Config karena daftar allowedJID adalah bagian dari konfigurasi.
func (c *Config) IsAllowedGroup(chatJID string) bool {
	for _, allowed := range c.AllowedGroupJIDs {
		if chatJID == allowed {
			return true
		}
	}
	return false
}
