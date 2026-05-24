package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"wa-gemini-bot/internal/config"
)

// ==========================================================================
// DokuService mengelola integrasi dengan DOKU Checkout API.
//
// Interface yang ditawarkan ke luar sangat kecil:
//   - CreatePayment()      → buat checkout session + track pending donasi
//   - SetPaymentCallback() → daftarkan callback saat pembayaran berhasil
//   - StartWebhookServer() → mulai HTTP server untuk terima notifikasi DOKU
//
// Semua detail (signature generation, HTTP request, pending map, mutex)
// tersembunyi di balik interface ini — sesuai prinsip "deep module".
//
// Alur end-to-end:
//   1. User kirim "donasi 50000" → bot panggil CreatePayment()
//   2. CreatePayment() → hit DOKU API, track pending, return URL
//   3. User bayar via checkout page
//   4. DOKU kirim webhook → WebhookHandler cocokkan dengan pending
//   5. Jika cocok → panggil onPaymentSuccess callback → bot kirim terima kasih
// ==========================================================================

// PaymentResult berisi informasi yang dibutuhkan caller setelah payment dibuat.
// Sengaja minimal — caller hanya perlu URL untuk dikirim ke user.
type PaymentResult struct {
	PaymentURL string
}

// DokuService membungkus semua interaksi dengan DOKU Checkout API.
// Field-field di-unexport karena merupakan detail implementasi.
type DokuService struct {
	clientID  string
	secretKey string
	baseURL   string

	// Pending donations — mapping invoice → donasi yang belum dibayar.
	// Diakses dari goroutine webhook handler, jadi perlu mutex.
	pending   map[string]*pendingDonation
	pendingMu sync.Mutex

	// Callback yang dipanggil saat pembayaran berhasil.
	// Didaftarkan oleh Bot via SetPaymentCallback.
	onPaymentSuccess func(chatJID, senderName, amount string)
}

// pendingDonation menyimpan konteks donasi yang belum dibayar.
// Unexported — detail internal tracking, bukan bagian dari public API.
type pendingDonation struct {
	invoiceNumber string
	senderName    string
	amount        string
	chatJID       string
	createdAt     time.Time
}

// NewDokuService membuat DokuService yang siap pakai.
// Menerima Config langsung sehingga caller (main.go) tidak perlu
// "membongkar" config ke struct perantara — mengurangi information leakage.
func NewDokuService(cfg *config.Config) *DokuService {
	baseURL := "https://api.doku.com"
	if cfg.DokuIsSandbox {
		baseURL = "https://api-sandbox.doku.com"
	}

	return &DokuService{
		clientID:  cfg.DokuClientID,
		secretKey: cfg.DokuSecretKey,
		baseURL:   baseURL,
		pending:   make(map[string]*pendingDonation),
	}
}

// SetPaymentCallback mendaftarkan fungsi yang dipanggil saat pembayaran berhasil.
// Didesain sebagai callback (bukan return channel) karena:
// - Satu payment → satu aksi, tidak perlu streaming
// - Callback bisa memanggil method Bot tanpa coupling balik ke DokuService
func (d *DokuService) SetPaymentCallback(fn func(chatJID, senderName, amount string)) {
	d.onPaymentSuccess = fn
}

// ==========================================================================
// CreatePayment — satu method yang menangani seluruh alur pembuatan pembayaran.
//
// Menggabungkan "buat checkout" dan "track pending" menjadi satu operasi
// atomik. Caller tidak perlu ingat urutan pemanggilan — ini menerapkan
// prinsip "define errors out of existence" (menghilangkan kemungkinan
// lupa memanggil track setelah create).
// ==========================================================================

// CreatePayment membuat checkout session DOKU dan langsung men-track
// donasi yang pending. Return PaymentResult yang berisi URL checkout.
func (d *DokuService) CreatePayment(invoiceNumber, amount, senderName, chatJID, notificationURL string) (*PaymentResult, error) {
	// 1. Buat checkout di DOKU
	paymentURL, err := d.requestCheckout(invoiceNumber, amount, notificationURL)
	if err != nil {
		return nil, err
	}

	// 2. Track pending — otomatis, caller tidak perlu panggil terpisah
	d.trackDonation(invoiceNumber, senderName, amount, chatJID)

	return &PaymentResult{PaymentURL: paymentURL}, nil
}

// ==========================================================================
// Internal — semua method di bawah ini unexported (private)
// ==========================================================================

// requestCheckout mengirim request ke DOKU Checkout API.
func (d *DokuService) requestCheckout(invoiceNumber, amount, notificationURL string) (string, error) {
	requestBody := map[string]interface{}{
		"order": map[string]interface{}{
			"amount":         parseAmount(amount),
			"invoice_number": invoiceNumber,
		},
		"payment": map[string]interface{}{
			"payment_due_date": 30,
		},
	}

	if notificationURL != "" {
		requestBody["additional_info"] = map[string]string{
			"override_notification_url": notificationURL,
		}
	}

	bodyBytes, _ := json.Marshal(requestBody)

	requestID := fmt.Sprintf("DONASI-%d", time.Now().UnixNano())
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	requestTarget := "/checkout/v1/payment"

	signature := d.generateSignature(requestID, timestamp, requestTarget, bodyBytes)

	req, err := http.NewRequest("POST", d.baseURL+requestTarget, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Client-Id", d.clientID)
	req.Header.Set("Request-Id", requestID)
	req.Header.Set("Request-Timestamp", timestamp)
	req.Header.Set("Signature", signature)

	log.Printf("[DOKU] Checkout request: %s", string(bodyBytes))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("gagal request checkout: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[DOKU] Checkout response (HTTP %d): %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("DOKU checkout gagal (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return extractPaymentURL(respBody)
}

// extractPaymentURL mengambil payment URL dari response JSON DOKU.
// Dipisah agar requestCheckout tidak terlalu panjang dan parsing
// bisa diuji secara independen.
func extractPaymentURL(respBody []byte) (string, error) {
	var parsed struct {
		Message  []string `json:"message"`
		Response struct {
			Payment struct {
				URL string `json:"url"`
			} `json:"payment"`
		} `json:"response"`
	}

	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("gagal parse response checkout: %w", err)
	}

	if parsed.Response.Payment.URL == "" {
		return "", fmt.Errorf("DOKU checkout: payment URL kosong, messages: %v", parsed.Message)
	}

	return parsed.Response.Payment.URL, nil
}

// trackDonation menyimpan donasi yang pending (menunggu pembayaran).
// Private — dipanggil otomatis oleh CreatePayment.
func (d *DokuService) trackDonation(invoiceNumber, senderName, amount, chatJID string) {
	d.pendingMu.Lock()
	defer d.pendingMu.Unlock()

	d.pending[invoiceNumber] = &pendingDonation{
		invoiceNumber: invoiceNumber,
		senderName:    senderName,
		amount:        amount,
		chatJID:       chatJID,
		createdAt:     time.Now(),
	}
	log.Printf("[DOKU] Donasi tracked: %s dari %s sebesar Rp %s", invoiceNumber, senderName, amount)
}

// ==========================================================================
// Webhook — menerima HTTP Notification dari DOKU
// ==========================================================================

// StartWebhookServer memulai HTTP server untuk menerima notifikasi pembayaran.
func (d *DokuService) StartWebhookServer(port string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/doku/webhook", d.handleWebhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	log.Printf("[DOKU] Webhook server listening on port %s", port)
	if err := (&http.Server{Addr: ":" + port, Handler: mux}).ListenAndServe(); err != nil {
		log.Printf("[DOKU] Webhook server error: %v", err)
	}
}

// HandleWebhook memproses notifikasi pembayaran dari DOKU (public wrapper).
func (d *DokuService) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	d.handleWebhook(w, r)
}

// handleWebhook memproses notifikasi pembayaran dari DOKU.
// Mencocokkan invoice number dengan pending donations, lalu trigger callback.
func (d *DokuService) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	log.Printf("[DOKU] Webhook received: %s", string(body))

	// Parse notification — gunakan struct minimal, field lain diabaikan.
	// DOKU bisa menambah field baru kapan saja, jadi kita parse non-strict.
	var notif struct {
		Order struct {
			InvoiceNumber string `json:"invoice_number"`
		} `json:"order"`
		Transaction struct {
			Status string `json:"status"`
		} `json:"transaction"`
	}

	if err := json.Unmarshal(body, &notif); err != nil {
		log.Printf("[DOKU] Webhook parse error: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if notif.Transaction.Status == "SUCCESS" {
		d.handleSuccessfulPayment(notif.Order.InvoiceNumber)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleSuccessfulPayment mencocokkan invoice dengan pending donation
// dan memanggil callback jika ditemukan. Dipisah dari handleWebhook
// agar webhook handler tetap fokus pada HTTP concern.
func (d *DokuService) handleSuccessfulPayment(invoiceNumber string) {
	d.pendingMu.Lock()
	donation, ok := d.pending[invoiceNumber]
	if ok {
		delete(d.pending, invoiceNumber)
	}
	d.pendingMu.Unlock()

	if !ok {
		log.Printf("[DOKU] Webhook: invoice %s tidak ditemukan di pending", invoiceNumber)
		return
	}

	log.Printf("[DOKU] Pembayaran berhasil: %s dari %s sebesar Rp %s",
		invoiceNumber, donation.senderName, donation.amount)

	if d.onPaymentSuccess != nil {
		d.onPaymentSuccess(donation.chatJID, donation.senderName, donation.amount)
	}
}

// ==========================================================================
// Crypto — HMAC-SHA256 signature untuk DOKU Checkout
//
// Sesuai docs DOKU (non-SNAP):
//   1. Digest = Base64(SHA256(requestBody))
//   2. Components = "Client-Id:{id}\nRequest-Id:{rid}\n..." (newline-separated)
//   3. Signature = "HMACSHA256=" + Base64(HMAC-SHA256(secretKey, components))
// ==========================================================================

func (d *DokuService) generateSignature(requestID, timestamp, requestTarget string, body []byte) string {
	bodyHash := sha256.Sum256(body)
	digest := base64.StdEncoding.EncodeToString(bodyHash[:])

	// Newline-separated, TANPA trailing newline — sesuai spesifikasi DOKU
	components := fmt.Sprintf("Client-Id:%s\nRequest-Id:%s\nRequest-Timestamp:%s\nRequest-Target:%s\nDigest:%s",
		d.clientID, requestID, timestamp, requestTarget, digest,
	)

	mac := hmac.New(sha256.New, []byte(d.secretKey))
	mac.Write([]byte(components))
	return "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// ==========================================================================
// Utility
// ==========================================================================

// parseAmount mengonversi string angka menjadi int.
// Mengabaikan karakter non-digit sehingga "50.000" → 50000.
func parseAmount(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
