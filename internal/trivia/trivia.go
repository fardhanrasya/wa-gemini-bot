package trivia

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"wa-gemini-bot/internal/ai"
)

// TriviaService mengelola kuis trivia otomatis di grup WhatsApp.
//
// Setiap interval tertentu, bot mengirim pertanyaan trivia yang di-generate
// oleh Gemini AI ke semua grup. Member menjawab dengan mengetik A/B/C/D.
// Setelah timeout, bot mengumumkan jawaban benar dan scoreboard.
//
// Mengikuti pola yang sama dengan DokuService — menggunakan callback
// agar tidak bergantung langsung ke Bot.
type TriviaService struct {
	ai       *ai.AIService
	interval time.Duration
	timeout  time.Duration
	groups   []string

	mu     sync.Mutex
	active map[string]*activeTrivia // groupJID -> kuis aktif

	// Callbacks — diset oleh Bot via SetCallbacks
	onSendMessage  func(groupJID, text string)
	onSendPoll     func(groupJID, question string, options []string) (string, error)
	onRecordMemory func(groupJID, sender, text string)

	stopChan chan struct{}
}

// activeTrivia menyimpan state kuis yang sedang berlangsung di satu grup.
type activeTrivia struct {
	pollMessageID string
	question      string
	options       [4]string
	optionHashes  [4]string         // SHA256 hashes of options
	correctAnswer int               // Index 0-3 (A=0, B=1, C=2, D=3)
	answers       map[string]int    // senderName -> selectedIndex
	timer         *time.Timer
}

// triviaQuestion adalah format JSON dari Gemini untuk satu soal trivia.
type triviaQuestion struct {
	Question string `json:"question"`
	A        string `json:"a"`
	B        string `json:"b"`
	C        string `json:"c"`
	D        string `json:"d"`
	Answer   string `json:"answer"`
}

var triviaTopics = []string{
	"teknologi dan IT",
	"geografi dunia",
	"sejarah dunia",
	"sains dan alam",
	"pengetahuan umum",
	"budaya pop dan entertainment",
	"olahraga",
	"kuliner dunia",
	"matematika dasar",
	"bahasa dan sastra",
}

// NewTriviaService membuat TriviaService baru.
func NewTriviaService(ai *ai.AIService, groups []string, intervalMinutes, timeoutSeconds int) *TriviaService {
	return &TriviaService{
		ai:       ai,
		interval: time.Duration(intervalMinutes) * time.Minute,
		timeout:  time.Duration(timeoutSeconds) * time.Second,
		groups:   groups,
		active:   make(map[string]*activeTrivia),
		stopChan: make(chan struct{}),
	}
}

// SetCallbacks mendaftarkan fungsi callback untuk mengirim pesan dan merekam memori.
func (t *TriviaService) SetCallbacks(
	sendMsg func(groupJID, text string),
	sendPoll func(groupJID, question string, options []string) (string, error),
	recordMem func(groupJID, sender, text string),
) {
	t.onSendMessage = sendMsg
	t.onSendPoll = sendPoll
	t.onRecordMemory = recordMem
}

// Start memulai loop trivia otomatis di background goroutine.
func (t *TriviaService) Start() {
	log.Printf("[TRIVIA] Started! Interval: %v, Timeout jawaban: %v", t.interval, t.timeout)

	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.sendTriviaToAllGroups()
			case <-t.stopChan:
				return
			}
		}
	}()
}

// Stop menghentikan loop trivia.
func (t *TriviaService) Stop() {
	close(t.stopChan)
}

// IsActive mengembalikan true jika ada kuis aktif di grup ini.
func (t *TriviaService) IsActive(groupJID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.active[groupJID]
	return ok
}

// RecordAnswer mencatat jawaban user dari polling berdasarkan hash pilihan.
func (t *TriviaService) RecordAnswer(groupJID, pollMsgID, senderName string, selectedHashes [][]byte) {
	t.mu.Lock()
	defer t.mu.Unlock()

	quiz, ok := t.active[groupJID]
	if !ok || quiz.pollMessageID != pollMsgID {
		return
	}

	if len(selectedHashes) == 0 {
		delete(quiz.answers, senderName)
		return
	}

	// Cocokkan hash yang diterima dengan hash opsi kita
	// WhatsApp mengirim hash dari pilihan
	for _, hash := range selectedHashes {
		hashStr := fmt.Sprintf("%X", hash)
		for i, h := range quiz.optionHashes {
			if h == hashStr {
				quiz.answers[senderName] = i
				log.Printf("[TRIVIA] %s memilih opsi %d (%s) di grup %s", senderName, i, quiz.options[i], groupJID)
				return
			}
		}
	}
}

// ==========================================================================
// Internal
// ==========================================================================

// maxRetries adalah jumlah percobaan ulang saat generateQuestion gagal.
// Gemini API bisa gagal sementara (rate limit, timeout, JSON invalid),
// jadi retry dengan backoff mencegah hilangnya satu ronde trivia.
const maxRetries = 3

func (t *TriviaService) sendTriviaToAllGroups() {
	// Guard: jangan kirim soal baru jika masih ada kuis aktif yang belum selesai.
	// Tanpa guard ini, soal baru akan menimpa kuis lama (timer di-Stop),
	// sehingga jawaban dan hasil kuis lama tidak pernah terkirim.
	t.mu.Lock()
	hasActive := len(t.active) > 0
	t.mu.Unlock()
	if hasActive {
		log.Printf("[TRIVIA] Masih ada kuis aktif yang belum selesai, skip ronde ini")
		return
	}

	log.Printf("[TRIVIA] Ticker fired — generating new question...")

	question, err := t.generateQuestionWithRetry()
	if err != nil {
		log.Printf("[TRIVIA] ❌ Gagal generate pertanyaan setelah %d percobaan: %v", maxRetries, err)
		return
	}

	log.Printf("[TRIVIA] ✅ Pertanyaan berhasil: %s", question.Question)

	for _, groupJID := range t.groups {
		t.startQuiz(groupJID, question)
	}
}

// generateQuestionWithRetry mencoba generate pertanyaan hingga maxRetries kali.
// Backoff eksponensial (2s, 4s, 6s) memberikan jeda agar rate limit mereda.
func (t *TriviaService) generateQuestionWithRetry() (*triviaQuestion, error) {
	var lastErr error
	for attempt := range maxRetries {
		q, err := t.generateQuestion()
		if err == nil {
			return q, nil
		}
		lastErr = err
		log.Printf("[TRIVIA] Generate gagal (attempt %d/%d): %v", attempt+1, maxRetries, err)
		time.Sleep(time.Duration(attempt+1) * 2 * time.Second)
	}
	return nil, lastErr
}

func (t *TriviaService) generateQuestion() (*triviaQuestion, error) {
	topic := triviaTopics[rand.Intn(len(triviaTopics))]

	prompt := fmt.Sprintf(`Buatkan 1 pertanyaan kuis trivia tentang "%s".

PENTING: Balas HANYA dengan JSON berikut, TANPA markdown, TANPA backtick, TANPA penjelasan:
{"question":"Pertanyaan di sini?","a":"Opsi A","b":"Opsi B","c":"Opsi C","d":"Opsi D","answer":"X"}

Aturan:
- Pertanyaan dalam Bahasa Indonesia
- Menarik, tidak terlalu mudah tapi juga tidak terlalu sulit
- Hanya 1 jawaban benar
- Field "answer" HARUS huruf kapital: A, B, C, atau D
- Opsi jawaban singkat (maks 50 karakter)`, topic)

	resp, err := t.ai.Ask(prompt)
	if err != nil {
		return nil, fmt.Errorf("AI error: %w", err)
	}

	resp = cleanJSONResponse(resp)

	var q triviaQuestion
	if err := json.Unmarshal([]byte(resp), &q); err != nil {
		return nil, fmt.Errorf("gagal parse trivia JSON: %w (response: %s)", err, resp)
	}

	q.Answer = strings.ToUpper(strings.TrimSpace(q.Answer))
	// Validasi dan konversi ke index
	validAnswers := map[string]int{"A": 0, "B": 1, "C": 2, "D": 3}
	_, ok := validAnswers[q.Answer]
	if !ok {
		return nil, fmt.Errorf("jawaban tidak valid: %s", q.Answer)
	}

	if q.Question == "" || q.A == "" || q.B == "" || q.C == "" || q.D == "" {
		return nil, fmt.Errorf("trivia question incomplete")
	}

	return &q, nil
}

// cleanJSONResponse membersihkan response dari markdown code blocks.
func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func (t *TriviaService) startQuiz(groupJID string, q *triviaQuestion) {
	options := []string{q.A, q.B, q.C, q.D}

	// Konversi A/B/C/D ke index 0-3
	ansMap := map[string]int{"A": 0, "B": 1, "C": 2, "D": 3}
	correctIdx := ansMap[q.Answer]

	var pollMsgID string
	var err error
	if t.onSendPoll != nil {
		question := fmt.Sprintf("🧠 %s (Batas waktu: %d detik)", q.Question, int(t.timeout.Seconds()))
		pollMsgID, err = t.onSendPoll(groupJID, question, options)
	}

	if err != nil || pollMsgID == "" {
		log.Printf("[TRIVIA] Gagal kirim polling ke %s: %v", groupJID, err)
		return
	}

	t.mu.Lock()
	// Jika masih ada kuis aktif (seharusnya tidak terjadi berkat guard di
	// sendTriviaToAllGroups, tapi defensive), reveal dulu sebelum replace.
	if old, ok := t.active[groupJID]; ok {
		old.timer.Stop()
		delete(t.active, groupJID)
		log.Printf("[TRIVIA] ⚠️ Kuis lama di %s ditimpa — seharusnya tidak terjadi", groupJID)
	}

	var hashes [4]string
	for i, opt := range options {
		h := sha256.Sum256([]byte(opt))
		hashes[i] = fmt.Sprintf("%X", h[:])
	}

	quiz := &activeTrivia{
		pollMessageID: pollMsgID,
		question:      q.Question,
		options:       [4]string{q.A, q.B, q.C, q.D},
		optionHashes:  hashes,
		correctAnswer: correctIdx,
		answers:       make(map[string]int),
	}
	t.active[groupJID] = quiz
	t.mu.Unlock()

	if t.onRecordMemory != nil {
		t.onRecordMemory(groupJID, "Abdul (Bot)", fmt.Sprintf("[Trivia Poll: %s]", q.Question))
	}

	log.Printf("[TRIVIA] Kuis dimulai di %s: %s (jawaban index: %d)", groupJID, q.Question, correctIdx)

	quiz.timer = time.AfterFunc(t.timeout, func() {
		t.revealAnswer(groupJID)
	})
}

func (t *TriviaService) revealAnswer(groupJID string) {
	t.mu.Lock()
	quiz, ok := t.active[groupJID]
	if !ok {
		t.mu.Unlock()
		return
	}
	delete(t.active, groupJID)
	t.mu.Unlock()

	labels := []string{"A", "B", "C", "D"}
	correctLetter := labels[quiz.correctAnswer]
	correctText := quiz.options[quiz.correctAnswer]

	var sb strings.Builder
	sb.WriteString("⏰ *WAKTU HABIS!*\n\n")
	sb.WriteString(fmt.Sprintf("✅ Jawaban benar: *%s) %s*\n\n", correctLetter, correctText))

	var correct, wrong []string
	for name, ansIdx := range quiz.answers {
		if ansIdx == quiz.correctAnswer {
			correct = append(correct, name)
		} else {
			wrong = append(wrong, fmt.Sprintf("%s (pilih: %s)", name, labels[ansIdx]))
		}
	}

	if len(quiz.answers) == 0 {
		sb.WriteString("😴 Tidak ada yang menjawab!\n")
	} else {
		if len(correct) > 0 {
			sb.WriteString("🏆 *Jawaban Benar:*\n")
			for i, name := range correct {
				sb.WriteString(fmt.Sprintf("  %d. %s ✅\n", i+1, name))
			}
		}
		if len(wrong) > 0 {
			sb.WriteString("\n❌ *Jawaban Salah:*\n")
			for i, info := range wrong {
				sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, info))
			}
		}
		sb.WriteString(fmt.Sprintf("\n📊 Skor: %d/%d benar", len(correct), len(quiz.answers)))
	}

	if t.onSendMessage != nil {
		t.onSendMessage(groupJID, sb.String())
	}

	log.Printf("[TRIVIA] Kuis selesai di %s: %d benar, %d salah", groupJID, len(correct), len(wrong))
}
