package trading

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"wa-gemini-bot/internal/economy"
)

// WebHandler menjembatani HTTP Requests ke TradingService.
type WebHandler struct {
	service *TradingService
}

// NewWebHandler membuat instance WebHandler baru.
func NewWebHandler(service *TradingService) *WebHandler {
	return &WebHandler{service: service}
}

// ─────────────────────────────────────────────────────────────
// AUTH MIDDLEWARE
// ─────────────────────────────────────────────────────────────

func (h *WebHandler) getSessionJID(r *http.Request) (string, error) {
	cookie, err := r.Cookie("trading_session")
	if err != nil {
		return "", ErrTradingSessionExpired
	}
	return h.service.VerifySession(cookie.Value)
}

// writeJSON menulis response JSON ke client.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError menulis error JSON ke client.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ─────────────────────────────────────────────────────────────
// PAGE ROUTES
// ─────────────────────────────────────────────────────────────

// HandleDashboard menyajikan halaman trading dashboard utama.
func (h *WebHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	_, err := h.getSessionJID(r)
	if err != nil {
		http.Redirect(w, r, "/trading/login-required", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(DashboardHTML))
}

// HandleLoginRequired menyajikan halaman instruksi login.
func (h *WebHandler) HandleLoginRequired(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(LoginRequiredHTML))
}

// HandleLogin memproses login satu-kali berbasis token dari magic link WhatsApp.
func (h *WebHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token login kosong.", http.StatusBadRequest)
		return
	}

	jid, err := h.service.VerifyLoginToken(token)
	if err != nil {
		http.Error(w, "Tautan login tidak valid atau sudah kedaluwarsa. Ketik '@bot trading' di WhatsApp untuk tautan baru.", http.StatusUnauthorized)
		return
	}

	sessionID, errSession := h.service.CreateSession(jid)
	if errSession != nil {
		http.Error(w, "Gagal membuat sesi login.", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "trading_session",
		Value:    sessionID,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("[TRADING] 🔓 LOGIN: %s berhasil masuk", jid)
	http.Redirect(w, r, "/trading", http.StatusFound)
}

// ─────────────────────────────────────────────────────────────
// API ROUTER
// ─────────────────────────────────────────────────────────────

// HandleAPI routes semua /trading/api/* requests.
func (h *WebHandler) HandleAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch path {
	// Account
	case "/trading/api/status":
		h.handleAPIStatus(w, r)
	case "/trading/api/deposit":
		h.handleAPIDeposit(w, r)
	case "/trading/api/withdraw":
		h.handleAPIWithdraw(w, r)
	case "/trading/api/logout":
		h.handleAPILogout(w, r)

	// Trading
	case "/trading/api/session/start":
		h.handleAPIStartSession(w, r)
	case "/trading/api/session/resolution":
		h.handleAPIResolution(w, r)
	case "/trading/api/session/check-stops":
		h.handleAPICheckStops(w, r)
	case "/trading/api/session/end":
		h.handleAPIEndSession(w, r)
	case "/trading/api/position/open":
		h.handleAPIOpenPosition(w, r)
	case "/trading/api/position/close":
		h.handleAPIClosePosition(w, r)
	case "/trading/api/position/stop-loss":
		h.handleAPISetStopLoss(w, r)
	case "/trading/api/position/take-profit":
		h.handleAPISetTakeProfit(w, r)
	case "/trading/api/position/trailing-stop":
		h.handleAPISetTrailingStop(w, r)

	// History
	case "/trading/api/history/trades":
		h.handleAPITradeHistory(w, r)
	case "/trading/api/history/sessions":
		h.handleAPISessionHistory(w, r)
	case "/trading/api/leaderboard":
		h.handleAPILeaderboard(w, r)

	// Tutorial
	case "/trading/api/tutorial/progress":
		h.handleAPITutorialProgress(w, r)
	case "/trading/api/tutorial/complete-step":
		h.handleAPITutorialComplete(w, r)
	case "/trading/api/tutorial/practice":
		h.handleAPIPractice(w, r)

	// Debug
	case "/trading/api/debug/auth":
		h.handleDebugAuth(w, r)
	case "/trading/api/debug/set-balance":
		h.handleDebugSetBalance(w, r)
	case "/trading/api/debug/patterns":
		h.handleDebugPatterns(w, r)
	case "/trading/api/debug/reveal":
		h.handleDebugReveal(w, r)

	default:
		writeError(w, http.StatusNotFound, "endpoint tidak ditemukan")
	}
}

// ─────────────────────────────────────────────────────────────
// ACCOUNT APIs
// ─────────────────────────────────────────────────────────────

func (h *WebHandler) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	acc, errAcc := h.service.GetTradingBalance(jid)
	if errAcc != nil {
		writeError(w, http.StatusInternalServerError, errAcc.Error())
		return
	}

	stats, _ := h.service.GetTradingStats(jid)

	// Ambil balance economy untuk max leverage
	chipBalance, _ := h.service.eco.GetBalance(jid)
	rank := economy.GetRankByBalance(chipBalance)
	maxLev := MaxLeverageForBalance(chipBalance)

	// Cek sesi aktif
	session, hasActive := h.service.GetActiveSession(jid)
	var activeSessionData map[string]interface{}
	if hasActive {
		// Hitung tick index yang sedang berjalan berdasarkan elapsed time
		elapsed := time.Since(session.StartTime).Seconds()
		// Resolution ticks mulai setelah ~4 detik delay di client
		tickIndex := int((elapsed - 4) / 3)
		if tickIndex < 0 {
			tickIndex = 0
		}
		if tickIndex > resolutionCandles {
			tickIndex = resolutionCandles
		}

		// Reveal resolution candles up to tickIndex
		revealedResolution := []PricePoint{}
		if tickIndex > 0 {
			if tickIndex > len(session.Chart.ResolutionData) {
				revealedResolution = session.Chart.ResolutionData
			} else {
				revealedResolution = session.Chart.ResolutionData[:tickIndex]
			}
		}

		activeSessionData = map[string]interface{}{
			"session_id":          session.ID,
			"observation":         session.Chart.ObservationData,
			"revealed_resolution": revealedResolution,
			"tick_index":          tickIndex,
			"news":                session.Chart.NewsEvents,
			"indicators":          session.Chart.Indicators,
			"difficulty":          session.Chart.Difficulty,
			"duration":            totalCandles,
			"obs_candles":         observationCandles,
			"res_candles":         resolutionCandles,
			"positions":           session.Positions,
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"account":            acc,
		"stats":              stats,
		"chip_balance":       chipBalance,
		"rank_name":          rank.Name,
		"rank_emoji":         rank.Emoji,
		"rank_styled":        rank.Styled,
		"max_leverage":       maxLev,
		"has_active_session": hasActive,
		"active_session":     activeSessionData,
	})
}

func (h *WebHandler) handleAPIDeposit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	var req struct {
		Amount int `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.service.Deposit(jid, req.Amount); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "deposited": req.Amount})
}

func (h *WebHandler) handleAPIWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	var req struct {
		Amount int `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.service.Withdraw(jid, req.Amount); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "withdrawn": req.Amount})
}

func (h *WebHandler) handleAPILogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("trading_session")
	if err == nil {
		h.service.DestroySession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "trading_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ─────────────────────────────────────────────────────────────
// TRADING APIs
// ─────────────────────────────────────────────────────────────

func (h *WebHandler) handleAPIStartSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	chart, sessionID, errStart := h.service.StartSession(jid)
	if errStart != nil {
		writeError(w, http.StatusInternalServerError, errStart.Error())
		return
	}

	// Kirim observation data saja (resolution hidden)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"session_id":  sessionID,
		"observation": chart.ObservationData,
		"news":        chart.NewsEvents,
		"indicators":  chart.Indicators,
		"difficulty":  chart.Difficulty,
		"duration":    totalCandles,
		"obs_candles": observationCandles,
		"res_candles": resolutionCandles,
	})
}

func (h *WebHandler) handleAPIResolution(w http.ResponseWriter, r *http.Request) {
	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	sessionIDStr := r.URL.Query().Get("id")
	sessionID, _ := strconv.Atoi(sessionIDStr)

	resolution, indicators, patternName, errRes := h.service.GetResolutionData(jid, sessionID)
	if errRes != nil {
		writeError(w, http.StatusBadRequest, errRes.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"resolution":   resolution,
		"indicators":   indicators,
		"pattern_name": patternName,
	})
}

func (h *WebHandler) handleAPIEndSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	errEnd := h.service.EndSession(jid)
	if errEnd != nil {
		writeError(w, http.StatusBadRequest, errEnd.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true})
}

func (h *WebHandler) handleAPICheckStops(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	var req struct {
		Price float64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if !isValidPrice(req.Price) {
		writeError(w, http.StatusBadRequest, ErrInvalidPrice.Error())
		return
	}

	triggered := h.service.CheckStopOrders(jid, req.Price)
	writeJSON(w, http.StatusOK, map[string]interface{}{"triggered": triggered})
}

func (h *WebHandler) handleAPIOpenPosition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	var req OpenPositionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	pos, errOpen := h.service.OpenPosition(jid, req)
	if errOpen != nil {
		writeError(w, http.StatusBadRequest, errOpen.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "position": pos})
}

func (h *WebHandler) handleAPIClosePosition(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	var req struct {
		PositionID int     `json:"position_id"`
		ExitPrice  float64 `json:"exit_price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if !isValidPrice(req.ExitPrice) {
		writeError(w, http.StatusBadRequest, ErrInvalidPrice.Error())
		return
	}

	pos, errClose := h.service.ClosePosition(jid, req.PositionID, req.ExitPrice)
	if errClose != nil {
		writeError(w, http.StatusBadRequest, errClose.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "position": pos})
}

func (h *WebHandler) handleAPISetStopLoss(w http.ResponseWriter, r *http.Request) {
	h.handleSetOrderField(w, r, "stop_loss")
}

func (h *WebHandler) handleAPISetTakeProfit(w http.ResponseWriter, r *http.Request) {
	h.handleSetOrderField(w, r, "take_profit")
}

func (h *WebHandler) handleAPISetTrailingStop(w http.ResponseWriter, r *http.Request) {
	h.handleSetOrderField(w, r, "trailing_stop")
}

func (h *WebHandler) handleSetOrderField(w http.ResponseWriter, r *http.Request, field string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	var req struct {
		PositionID int     `json:"position_id"`
		Value      float64 `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	var errSet error
	switch field {
	case "stop_loss":
		errSet = h.service.SetStopLoss(jid, req.PositionID, req.Value)
	case "take_profit":
		errSet = h.service.SetTakeProfit(jid, req.PositionID, req.Value)
	case "trailing_stop":
		errSet = h.service.SetTrailingStop(jid, req.PositionID, req.Value)
	}

	if errSet != nil {
		writeError(w, http.StatusBadRequest, errSet.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// ─────────────────────────────────────────────────────────────
// HISTORY APIs
// ─────────────────────────────────────────────────────────────

func (h *WebHandler) handleAPITradeHistory(w http.ResponseWriter, r *http.Request) {
	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 20
	if n, e := strconv.Atoi(limitStr); e == nil && n > 0 {
		limit = n
	}

	records, errHist := h.service.GetTradeHistory(jid, limit)
	if errHist != nil {
		writeError(w, http.StatusInternalServerError, errHist.Error())
		return
	}

	writeJSON(w, http.StatusOK, records)
}

func (h *WebHandler) handleAPISessionHistory(w http.ResponseWriter, r *http.Request) {
	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if n, e := strconv.Atoi(limitStr); e == nil && n > 0 {
		limit = n
	}

	sessions, errHist := h.service.GetSessionHistory(jid, limit)
	if errHist != nil {
		writeError(w, http.StatusInternalServerError, errHist.Error())
		return
	}

	writeJSON(w, http.StatusOK, sessions)
}

func (h *WebHandler) handleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	// Ambil Top 20 trader teratas secara global
	list, err := h.service.GetLeaderboard(20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// ─────────────────────────────────────────────────────────────
// TUTORIAL APIs
// ─────────────────────────────────────────────────────────────

func (h *WebHandler) handleAPITutorialProgress(w http.ResponseWriter, r *http.Request) {
	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	progress, errProg := h.service.GetTutorialProgress(jid)
	if errProg != nil {
		writeError(w, http.StatusInternalServerError, errProg.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"progress": progress,
		"steps":    TutorialSteps,
		"complete": IsTutorialComplete(progress),
	})
}

func (h *WebHandler) handleAPITutorialComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	var req struct {
		Step int `json:"step"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := h.service.CompleteTutorialStep(jid, req.Step); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *WebHandler) handleAPIPractice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	chart, errPractice := h.service.StartPracticeSession(jid)
	if errPractice != nil {
		writeError(w, http.StatusInternalServerError, errPractice.Error())
		return
	}

	// Mark practice as done
	_ = h.service.CompletePractice(jid)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"observation": chart.ObservationData,
		"resolution":  chart.ResolutionData,
		"news":        chart.NewsEvents,
		"indicators":  chart.Indicators,
		"pattern":     chart.PatternName,
		"direction":   chart.ExpectedDir,
		"practice":    true,
	})
}

// ─────────────────────────────────────────────────────────────
// DEBUG APIs
// ─────────────────────────────────────────────────────────────

func (h *WebHandler) handleDebugAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if !h.service.ValidateDebugPassword(req.Password) {
		writeError(w, http.StatusUnauthorized, "Password debug salah")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *WebHandler) handleDebugSetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Password string `json:"password"`
		Amount   int    `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if !h.service.ValidateDebugPassword(req.Password) {
		writeError(w, http.StatusUnauthorized, "Password debug salah")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	if err := h.service.DebugSetBalance(jid, req.Amount); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "new_balance": req.Amount})
}

func (h *WebHandler) handleDebugPatterns(w http.ResponseWriter, r *http.Request) {
	pwd := r.URL.Query().Get("password")
	if !h.service.ValidateDebugPassword(pwd) {
		writeError(w, http.StatusUnauthorized, "Password debug salah")
		return
	}

	writeJSON(w, http.StatusOK, h.service.DebugGetPatternList())
}

func (h *WebHandler) handleDebugReveal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Password  string `json:"password"`
		SessionID int    `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if !h.service.ValidateDebugPassword(req.Password) {
		writeError(w, http.StatusUnauthorized, "Password debug salah")
		return
	}

	jid, err := h.getSessionJID(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "Sesi habis")
		return
	}

	resolution, indicators, patternName, errRes := h.service.GetResolutionData(jid, req.SessionID)
	if errRes != nil {
		writeError(w, http.StatusBadRequest, errRes.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"resolution":   resolution,
		"indicators":   indicators,
		"pattern_name": patternName,
		"debug":        true,
	})
}

// ─────────────────────────────────────────────────────────────
// LOGIN REQUIRED PAGE (simple HTML)
// ─────────────────────────────────────────────────────────────

const LoginRequiredHTML = `<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Trading Simulator — Login Required</title>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;600;700&display=swap" rel="stylesheet">
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Plus Jakarta Sans', sans-serif;
            background: #090d16;
            color: #f3f4f6;
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
        }
        .card {
            background: #111827;
            border: 1px solid #374151;
            border-radius: 20px;
            padding: 3rem;
            text-align: center;
            max-width: 480px;
        }
        .icon { font-size: 4rem; margin-bottom: 1rem; }
        h1 { font-size: 1.5rem; margin-bottom: 0.75rem; }
        p { color: #9ca3af; line-height: 1.6; }
        code {
            background: rgba(139, 92, 246, 0.15);
            color: #8b5cf6;
            padding: 0.2rem 0.5rem;
            border-radius: 6px;
            font-weight: 600;
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="icon">🔒</div>
        <h1>Login Diperlukan</h1>
        <p>Untuk mengakses Trading Simulator, ketik <code>@bot trading</code> di grup WhatsApp untuk mendapatkan tautan login personal.</p>
    </div>
</body>
</html>`
