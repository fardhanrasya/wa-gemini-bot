package mining

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"wa-gemini-bot/internal/economy"
)

// WebHandler menjembatani HTTP Requests ke MiningService.
type WebHandler struct {
	service *MiningService
}

// NewWebHandler membuat instance WebHandler baru.
func NewWebHandler(service *MiningService) *WebHandler {
	return &WebHandler{
		service: service,
	}
}

// ==========================================================================
// AUTH MIDDLEWARE HELPER
// ==========================================================================

func (h *WebHandler) getSessionJID(r *http.Request) (string, error) {
	cookie, err := r.Cookie("mining_session")
	if err != nil {
		return "", ErrSessionExpired
	}

	jid, errVerify := h.service.VerifySession(cookie.Value)
	if errVerify != nil {
		return "", errVerify
	}

	return jid, nil
}

// ==========================================================================
// PAGE ROUTING HANDLERS
// ==========================================================================

// HandleDashboard menyajikan halaman dashboard penambangan utama (Memerlukan Cookie Sesi).
func (h *WebHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	jid, err := h.getSessionJID(r)
	if err != nil {
		// Jika sesi tidak valid, alihkan ke halaman dashboard publik
		http.Redirect(w, r, "/mining/public", http.StatusFound)
		return
	}

	_ = jid // Sesi valid, sajikan halaman dashboard personal
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(DashboardHTML))
}

// HandlePublicDashboard menyajikan halaman monitoring publik (Tanpa Login).
func (h *WebHandler) HandlePublicDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(PublicDashboardHTML))
}

// HandleLogin memproses login satu-kali berbasis token dari link WhatsApp.
func (h *WebHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		log.Printf("[MINING] ❌ LOGIN FAILED: Request login tanpa token")
		http.Error(w, "Token login kosong. Silakan minta tautan baru dari WhatsApp.", http.StatusBadRequest)
		return
	}

	// 1. Verifikasi token satu-kali di database
	jid, err := h.service.VerifyLoginToken(token)
	if err != nil {
		log.Printf("[MINING] ❌ LOGIN FAILED: Token '%s' tidak valid atau kedaluwarsa", token)
		http.Error(w, "Tautan login tidak valid atau sudah kedaluwarsa. Silakan ketik '@bot tambang' di WhatsApp untuk tautan baru.", http.StatusUnauthorized)
		return
	}

	// 2. Buat sesi baru untuk user ini
	sessionID, errSession := h.service.CreateSession(jid)
	if errSession != nil {
		log.Printf("[MINING] ❌ LOGIN ERROR: Gagal membuat sesi untuk %s: %v", jid, errSession)
		http.Error(w, "Gagal membuat sesi login.", http.StatusInternalServerError)
		return
	}

	// 3. Pasang secure Cookie di browser pemain
	http.SetCookie(w, &http.Cookie{
		Name:     "mining_session",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour), // Berlaku 7 hari
		HttpOnly: true,                               // Aman dari pencurian JavaScript (XSS)
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("[MINING] 🔓 LOGIN SUCCESS: JID %s berhasil masuk (Session: %s)", jid, sessionID[:6])

	// 4. Redirect ke dashboard penambangan personal
	http.Redirect(w, r, "/mining", http.StatusFound)
}

// ==========================================================================
// REST API ENDPOINTS
// ==========================================================================

// HandleAPILogout menghapus sesi aktif dan menghapus cookie dari browser.
func (h *WebHandler) HandleAPILogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("mining_session")
	if err == nil {
		h.service.DestroySession(cookie.Value)
	}

	// Hapus cookie dengan men-set masa kedaluwarsa ke masa lalu
	http.SetCookie(w, &http.Cookie{
		Name:     "mining_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"success":true}`))
}

// HandleAPIStatus mengembalikan data user, pangkat, dan rig aktif dalam format JSON.
func (h *WebHandler) HandleAPIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	jid, err := h.getSessionJID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Sesi habis atau tidak valid"}`))
		return
	}

	user, errUser := h.service.eco.GetUser(jid)
	if errUser != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal membaca profil pengguna"})
		return
	}

	rigs, errRigs := h.service.GetActiveRigs(jid)
	if errRigs != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal membaca rig penambangan"})
		return
	}

	rank := economy.GetRankByBalance(user.Balance)

	response := map[string]interface{}{
		"user":       user,
		"rank_name":  rank.Name,
		"rank_emoji": rank.Emoji,
		"rigs":       rigs,
	}

	json.NewEncoder(w).Encode(response)
}

// HandleAPIPublicList mengembalikan daftar publik rig milik semua pemain (read-only).
func (h *WebHandler) HandleAPIPublicList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	list, err := h.service.GetAllPlayersRigs()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal memuat leaderboard publik"})
		return
	}

	json.NewEncoder(w).Encode(list)
}

// HandleAPIClaim mencairkan chip hasil tambang dari semua rig aktif dan melakukan refuel bahan bakar.
func (h *WebHandler) HandleAPIClaim(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"Method not allowed"}`))
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Sesi habis"}`))
		return
	}

	claimedChips, errClaim := h.service.ClaimAndRefuel(jid)
	if errClaim != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": errClaim.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"claimed_chips": claimedChips,
	})
}

// HandleAPIBuy menyewa unit rig baru untuk pemain.
func (h *WebHandler) HandleAPIBuy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"error":"Method not allowed"}`))
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"Sesi habis"}`))
		return
	}

	var req struct {
		Tier string `json:"tier"`
	}

	errDecode := json.NewDecoder(r.Body).Decode(&req)
	if errDecode != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Invalid payload JSON"}`))
		return
	}

	errBuy := h.service.BuyRig(jid, req.Tier)
	if errBuy != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": errBuy.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	})
}
