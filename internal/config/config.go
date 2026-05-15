package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config menyimpan semua konfigurasi aplikasi.
//
// Semua validasi, default value, dan parsing environment variable ada di sini.
// Ini menerapkan prinsip "define errors out of existence" — caller yang
// menerima *Config dijamin mendapat state yang valid dan lengkap,
// tidak perlu melakukan validasi sendiri.
type Config struct {
	GeminiAPIKey     string
	GeminiModel      string
	SystemPrompt     string
	AllowedGroupJIDs []string
	MaxHistory       int

	// DOKU Checkout
	DokuClientID    string
	DokuSecretKey   string
	DokuIsSandbox   bool
	DokuWebhookPort string
	DokuWebhookURL  string // URL publik untuk notification (misal: https://xxx.ngrok-free.app/doku/webhook)
	DokuEnabled     bool   // true jika semua config DOKU terisi

	// Trivia Quiz
	TriviaEnabled          bool
	TriviaIntervalMinutes  int
	TriviaAnswerTimeoutSec int
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

	maxHistory := 20

	// DOKU Config — opsional. Jika tidak diisi, fitur donasi tidak aktif
	// tapi bot tetap berjalan normal. Ini menerapkan prinsip graceful
	// degradation — fitur tambahan tidak boleh merusak fungsi inti.
	dokuClientID := os.Getenv("DOKU_CLIENT_ID")
	dokuSecretKey := os.Getenv("DOKU_SECRET_KEY")
	dokuSandbox := os.Getenv("DOKU_SANDBOX")
	dokuWebhookPort := os.Getenv("DOKU_WEBHOOK_PORT")
	dokuWebhookURL := os.Getenv("DOKU_WEBHOOK_URL")

	if dokuWebhookPort == "" {
		dokuWebhookPort = "8080"
	}

	dokuEnabled := dokuClientID != "" && dokuSecretKey != ""

	// Trivia Config — opsional. Default: nonaktif.
	triviaEnabled := os.Getenv("TRIVIA_ENABLED") == "true" || os.Getenv("TRIVIA_ENABLED") == "1"
	triviaInterval := 30 // default 30 menit
	if v := os.Getenv("TRIVIA_INTERVAL_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			triviaInterval = n
		}
	}
	triviaTimeout := 1200 // default 1200 detik
	if v := os.Getenv("TRIVIA_ANSWER_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			triviaTimeout = n
		}
	}

	return &Config{
		GeminiAPIKey:     apiKey,
		GeminiModel:      model,
		SystemPrompt:     systemPrompt,
		AllowedGroupJIDs: allowedJIDs,
		MaxHistory:       maxHistory,

		DokuClientID:    dokuClientID,
		DokuSecretKey:   dokuSecretKey,
		DokuIsSandbox:   dokuSandbox == "" || dokuSandbox == "true" || dokuSandbox == "1",
		DokuWebhookPort: dokuWebhookPort,
		DokuWebhookURL:  dokuWebhookURL,
		DokuEnabled:     dokuEnabled,

		TriviaEnabled:          triviaEnabled,
		TriviaIntervalMinutes:  triviaInterval,
		TriviaAnswerTimeoutSec: triviaTimeout,
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
