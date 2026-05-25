package bot

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"go.mau.fi/whatsmeow/types"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"google.golang.org/protobuf/proto"

	"wa-gemini-bot/internal/mining"
)

// StartHTTPServer memulai HTTP server gabungan untuk webhook DOKU dan Admin Panel.
func (b *Bot) StartHTTPServer(port string) {
	mux := http.NewServeMux()

	// Webhook DOKU (jika diaktifkan)
	mux.HandleFunc("/doku/webhook", func(w http.ResponseWriter, r *http.Request) {
		if b.doku != nil {
			b.doku.HandleWebhook(w, r)
		} else {
			http.Error(w, "DOKU service is disabled", http.StatusNotFound)
		}
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Web Dashboard Pertambangan
	if b.mining != nil {
		handler := mining.NewWebHandler(b.mining)
		
		// Halaman UI
		mux.HandleFunc("/mining", handler.HandleDashboard)
		mux.HandleFunc("/mining/public", handler.HandlePublicDashboard)
		mux.HandleFunc("/mining/login", handler.HandleLogin)
		
		// API JSON
		mux.HandleFunc("/mining/api/status", handler.HandleAPIStatus)
		mux.HandleFunc("/mining/api/public-list", handler.HandleAPIPublicList)
		mux.HandleFunc("/mining/api/claim", handler.HandleAPIClaim)
		mux.HandleFunc("/mining/api/buy", handler.HandleAPIBuy)
		mux.HandleFunc("/mining/api/logout", handler.HandleAPILogout)
		
		log.Printf("[MINING] Web Dashboard routes registered successfully")
	}

	// Web Admin Panel UI & APIs (memerlukan ADMIN_PANEL_TOKEN)
	if b.config.AdminPanelToken == "" {
		log.Printf("[ADMIN] Admin panel dinonaktifkan — set ADMIN_PANEL_TOKEN di .env untuk mengaktifkan")
	} else {
		// Sajikan HTML secara publik agar browser dapat merender UI & memicu prompt login modal.
		// Keamanan tetap terjamin karena setiap REST API di bawah ini tetap di-wrap dengan withAdminAuth secara ketat.
		mux.HandleFunc("/admin", b.handleAdminPanelHTML)
		
		mux.HandleFunc("/admin/api/status", b.withAdminAuth(b.handleAdminAPIStatus))
		mux.HandleFunc("/admin/api/groups", b.withAdminAuth(b.handleAdminAPIGroups))
		mux.HandleFunc("/admin/api/broadcast", b.withAdminAuth(b.handleAdminAPIBroadcast))
		mux.HandleFunc("/admin/api/users", b.withAdminAuth(b.handleAdminAPIUsers))
		mux.HandleFunc("/admin/api/edit-balance", b.withAdminAuth(b.handleAdminAPIEditBalance))
	}

	log.Printf("[ADMIN] Unified HTTP Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Printf("[ADMIN] Unified HTTP Server error: %v", err)
	}
}

// withAdminAuth membungkus handler admin agar hanya dapat diakses dengan token yang valid.
func (b *Bot) withAdminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := b.config.AdminPanelToken
		if expected == "" {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Admin panel is not configured"}`, http.StatusServiceUnavailable)
			return
		}
		if !adminTokenMatches(r, expected) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", `Bearer realm="admin"`)
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func adminTokenMatches(r *http.Request, expected string) bool {
	got := adminTokenFromRequest(r)
	if got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

func adminTokenFromRequest(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	if t := strings.TrimSpace(r.Header.Get("X-Admin-Token")); t != "" {
		return t
	}
	return ""
}

// handleAdminAPIStatus mengembalikan status bot umum dalam format JSON.
func (b *Bot) handleAdminAPIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	status := map[string]interface{}{
		"status":          "online",
		"groups_count":    len(b.config.AllowedGroupJIDs),
		"poker_enabled":   b.config.PokerEnabled,
		"blackjack_enabled": b.config.BlackjackEnabled,
		"trivia_enabled":  b.config.TriviaEnabled,
		"upscale_enabled": b.cld != nil,
	}
	
	json.NewEncoder(w).Encode(status)
}

// handleAdminAPIGroups mengembalikan daftar grup yang diizinkan.
func (b *Bot) handleAdminAPIGroups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var groups []map[string]string
	for _, jid := range b.config.AllowedGroupJIDs {
		// Dapatkan nama grup dari whatsmeow store jika ada
		groupName := "Grup Chat WhatsApp"
		parsedJID, err := types.ParseJID(jid)
		if err == nil {
			info, errStore := b.client.Store.Contacts.GetContact(context.Background(), parsedJID)
			if errStore == nil && info.FullName != "" {
				groupName = info.FullName
			}
		}
		
		groups = append(groups, map[string]string{
			"jid":  jid,
			"name": groupName,
		})
	}
	
	json.NewEncoder(w).Encode(groups)
}

// handleAdminAPIBroadcast memproses request broadcast pesan ke satu atau semua grup.
func (b *Bot) handleAdminAPIBroadcast(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		GroupJID string `json:"group_jid"`
		Message  string `json:"message"`
	}
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Bad request"}`, http.StatusBadRequest)
		return
	}
	
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	
	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		http.Error(w, `{"error": "Message cannot be empty"}`, http.StatusBadRequest)
		return
	}
	
	var targets []string
	if req.GroupJID == "all" {
		targets = b.config.AllowedGroupJIDs
	} else {
		found := false
		for _, jid := range b.config.AllowedGroupJIDs {
			if jid == req.GroupJID {
				targets = append(targets, jid)
				found = true
				break
			}
		}
		if !found {
			http.Error(w, `{"error": "Invalid group JID"}`, http.StatusBadRequest)
			return
		}
	}
	
	successCount := 0
	for _, target := range targets {
		jid, err := types.ParseJID(target)
		if err != nil {
			log.Printf("[ADMIN] Gagal parse JID %s untuk broadcast: %v", target, err)
			continue
		}
		
		// Send message
		_, errSend := b.client.SendMessage(context.Background(), jid, &waProto.Message{
			Conversation: proto.String(req.Message),
		})
		if errSend != nil {
			log.Printf("[ADMIN] Gagal kirim broadcast ke %s: %v", target, errSend)
		} else {
			successCount++
		}
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Berhasil menyiarkan pesan ke %d grup.", successCount),
	})
}

// handleAdminAPIUsers mencari dan mengembalikan daftar user ekonomi.
func (b *Bot) handleAdminAPIUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	query := r.URL.Query().Get("q")
	users, err := b.eco.SearchUsers(query)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Gagal mengambil daftar user: %v"}`, err), http.StatusInternalServerError)
		return
	}
	
	json.NewEncoder(w).Encode(users)
}

// handleAdminAPIEditBalance memproses perubahan saldo user (set, add, subtract).
func (b *Bot) handleAdminAPIEditBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	
	var req struct {
		JID    string `json:"jid"`
		Amount int    `json:"amount"`
		Action string `json:"action"` // "set", "add", "subtract"
	}
	
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Bad request"}`, http.StatusBadRequest)
		return
	}
	
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}
	
	req.JID = strings.TrimSpace(req.JID)
	if req.JID == "" {
		http.Error(w, `{"error": "JID cannot be empty"}`, http.StatusBadRequest)
		return
	}
	if req.Amount < 0 {
		http.Error(w, `{"error": "Amount cannot be negative"}`, http.StatusBadRequest)
		return
	}
	
	var finalBalance int
	
	switch req.Action {
	case "set":
		if err := b.eco.SetBalance(req.JID, req.Amount, "admin_panel"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
			return
		}
		finalBalance = req.Amount
	case "add":
		if err := b.eco.AddBalance(req.JID, req.Amount, "admin_add", "admin_panel"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
			return
		}
		bal, _ := b.eco.GetBalance(req.JID)
		finalBalance = bal
	case "subtract":
		if err := b.eco.SubtractBalance(req.JID, req.Amount, "admin_subtract", "admin_panel"); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "%v"}`, err), http.StatusInternalServerError)
			return
		}
		bal, _ := b.eco.GetBalance(req.JID)
		finalBalance = bal
	default:
		http.Error(w, `{"error": "Invalid action, must be set, add, or subtract"}`, http.StatusBadRequest)
		return
	}
	
	user, errUser := b.eco.GetUser(req.JID)
	userName := req.JID
	if errUser == nil && user.Name != "" {
		userName = user.Name
	}
	
	log.Printf("[ADMIN] Saldo %s (%s) di-update via Admin Panel menjadi %d chip (Aksi: %s %d)", userName, req.JID, finalBalance, req.Action, req.Amount)
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"new_balance": finalBalance,
		"message": fmt.Sprintf("Berhasil memperbarui saldo %s menjadi %d chip.", userName, finalBalance),
	})
}

// handleAdminPanelHTML me-render halaman dashboard admin.
func (b *Bot) handleAdminPanelHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(adminPanelHTML))
}

const adminPanelHTML = `<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Abdul Bot — Dashboard Admin</title>
    <link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-base: #090d16;
            --bg-surface: #111827;
            --bg-panel: #1f2937;
            --text-primary: #f3f4f6;
            --text-secondary: #9ca3af;
            --accent-purple: #8b5cf6;
            --accent-purple-hover: #7c3aed;
            --accent-emerald: #10b981;
            --accent-rose: #ef4444;
            --border-color: #374151;
            --font-main: 'Plus Jakarta Sans', -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: var(--font-main);
            background-color: var(--bg-base);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            overflow-x: hidden;
        }

        /* HEADER */
        header {
            background-color: var(--bg-surface);
            border-bottom: 1px solid var(--border-color);
            padding: 1rem 2rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            position: sticky;
            top: 0;
            z-index: 10;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 0.75rem;
            font-size: 1.25rem;
            font-weight: 700;
            letter-spacing: -0.025em;
        }

        .logo-icon {
            font-size: 1.75rem;
            background: linear-gradient(135deg, var(--accent-purple), var(--accent-emerald));
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }

        .bot-status {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            font-size: 0.875rem;
            background: rgba(16, 185, 129, 0.1);
            color: var(--accent-emerald);
            padding: 0.375rem 0.75rem;
            border-radius: 9999px;
            font-weight: 600;
            border: 1px solid rgba(16, 185, 129, 0.2);
        }

        .status-dot {
            width: 8px;
            height: 8px;
            background-color: var(--accent-emerald);
            border-radius: 50%;
            display: inline-block;
            box-shadow: 0 0 8px var(--accent-emerald);
        }

        /* MAIN CONTENT */
        .container {
            display: flex;
            flex: 1;
            width: 100%;
            max-width: 1400px;
            margin: 0 auto;
            padding: 2rem;
            gap: 2rem;
        }

        /* SIDEBAR / TABS NAV */
        .sidebar {
            width: 250px;
            flex-shrink: 0;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        .nav-btn {
            background: none;
            border: none;
            color: var(--text-secondary);
            font-family: var(--font-main);
            font-size: 1rem;
            font-weight: 600;
            text-align: left;
            padding: 0.875rem 1.25rem;
            border-radius: 12px;
            cursor: pointer;
            display: flex;
            align-items: center;
            gap: 0.75rem;
            transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
        }

        .nav-btn:hover {
            background-color: rgba(255, 255, 255, 0.03);
            color: var(--text-primary);
        }

        .nav-btn.active {
            background-color: rgba(139, 92, 246, 0.1);
            color: var(--accent-purple);
            border-left: 3px solid var(--accent-purple);
            border-radius: 0 12px 12px 0;
        }

        /* CONTENT VIEWPORT */
        .main-content {
            flex: 1;
            display: flex;
            flex-direction: column;
            gap: 2rem;
            min-width: 0;
        }

        .tab-panel {
            display: none;
            animation: fadeIn 0.3s ease-out;
        }

        .tab-panel.active {
            display: flex;
            flex-direction: column;
            gap: 2rem;
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }

        /* CARDS / WIDGETS */
        .grid-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
            gap: 1.5rem;
        }

        .card-stat {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 16px;
            padding: 1.5rem;
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
            position: relative;
            overflow: hidden;
        }

        .card-stat::after {
            content: '';
            position: absolute;
            bottom: 0;
            left: 0;
            width: 100%;
            height: 4px;
            background: linear-gradient(90deg, var(--accent-purple), var(--accent-emerald));
        }

        .stat-label {
            font-size: 0.875rem;
            color: var(--text-secondary);
            font-weight: 500;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }

        .stat-value {
            font-size: 2.25rem;
            font-weight: 700;
            color: var(--text-primary);
        }

        /* PANELS */
        .panel {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            padding: 2rem;
            display: flex;
            flex-direction: column;
            gap: 1.5rem;
        }

        .panel-title {
            font-size: 1.25rem;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .panel-desc {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-top: -1rem;
        }

        /* FORM CONTROLS */
        .form-group {
            display: flex;
            flex-direction: column;
            gap: 0.5rem;
        }

        label {
            font-size: 0.875rem;
            font-weight: 600;
            color: var(--text-primary);
        }

        input[type="text"], input[type="number"], select, textarea {
            background-color: var(--bg-panel);
            border: 1px solid var(--border-color);
            border-radius: 12px;
            color: var(--text-primary);
            font-family: var(--font-main);
            font-size: 1rem;
            padding: 0.875rem 1.25rem;
            width: 100%;
            transition: all 0.2s;
        }

        input:focus, select:focus, textarea:focus {
            outline: none;
            border-color: var(--accent-purple);
            box-shadow: 0 0 0 2px rgba(139, 92, 246, 0.2);
        }

        textarea {
            resize: vertical;
            min-height: 120px;
        }

        .btn {
            background-color: var(--accent-purple);
            border: none;
            border-radius: 12px;
            color: white;
            cursor: pointer;
            font-family: var(--font-main);
            font-size: 1rem;
            font-weight: 600;
            padding: 0.875rem 1.75rem;
            transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
            display: inline-flex;
            justify-content: center;
            align-items: center;
            gap: 0.5rem;
        }

        .btn:hover {
            background-color: var(--accent-purple-hover);
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(139, 92, 246, 0.3);
        }

        .btn:active {
            transform: translateY(0);
        }

        .btn:disabled {
            background-color: var(--border-color);
            cursor: not-allowed;
            transform: none;
            box-shadow: none;
        }

        /* TABLES */
        .table-container {
            border: 1px solid var(--border-color);
            border-radius: 16px;
            overflow: hidden;
            width: 100%;
        }

        table {
            border-collapse: collapse;
            text-align: left;
            width: 100%;
            font-size: 0.9375rem;
        }

        th {
            background-color: rgba(255, 255, 255, 0.02);
            color: var(--text-secondary);
            font-weight: 600;
            padding: 1rem 1.5rem;
            border-bottom: 1px solid var(--border-color);
        }

        td {
            padding: 1rem 1.5rem;
            border-bottom: 1px solid var(--border-color);
            vertical-align: middle;
        }

        tr:last-child td {
            border-bottom: none;
        }

        tr:hover td {
            background-color: rgba(255, 255, 255, 0.01);
        }

        /* UTILITY ACCENTS */
        .chip-balance {
            display: inline-flex;
            align-items: center;
            background: rgba(16, 185, 129, 0.1);
            color: var(--accent-emerald);
            padding: 0.25rem 0.625rem;
            border-radius: 9999px;
            font-size: 0.8125rem;
            font-weight: 700;
            border: 1px solid rgba(16, 185, 129, 0.2);
        }

        .user-jid {
            font-family: monospace;
            font-size: 0.8125rem;
            color: var(--text-secondary);
            background-color: rgba(255, 255, 255, 0.03);
            padding: 0.25rem 0.5rem;
            border-radius: 6px;
        }

        .btn-edit {
            background-color: rgba(139, 92, 246, 0.1);
            color: var(--accent-purple);
            border: 1px solid rgba(139, 92, 246, 0.2);
            padding: 0.375rem 0.75rem;
            border-radius: 8px;
            font-size: 0.8125rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.2s;
        }

        .btn-edit:hover {
            background-color: var(--accent-purple);
            color: white;
            border-color: var(--accent-purple);
        }

        /* NOTIFICATIONS & TOASTS */
        .toast {
            position: fixed;
            bottom: 2rem;
            right: 2rem;
            background-color: var(--bg-surface);
            border-left: 4px solid var(--accent-purple);
            box-shadow: 0 10px 25px rgba(0,0,0,0.5);
            border-radius: 8px;
            padding: 1rem 1.5rem;
            display: flex;
            align-items: center;
            gap: 0.75rem;
            transform: translateY(150%);
            transition: transform 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            z-index: 100;
        }

        .toast.show {
            transform: translateY(0);
        }

        .toast.success {
            border-left-color: var(--accent-emerald);
        }

        .toast.error {
            border-left-color: var(--accent-rose);
        }

        /* MODAL */
        .modal-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.75);
            backdrop-filter: blur(4px);
            z-index: 50;
            display: flex;
            justify-content: center;
            align-items: center;
            opacity: 0;
            pointer-events: none;
            transition: opacity 0.2s;
        }

        .modal-overlay.show {
            opacity: 1;
            pointer-events: auto;
        }

        .modal {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            width: 100%;
            max-width: 500px;
            padding: 2rem;
            display: flex;
            flex-direction: column;
            gap: 1.5rem;
            transform: scale(0.95);
            transition: transform 0.2s;
            box-shadow: 0 20px 50px rgba(0, 0, 0, 0.6);
        }

        .modal-overlay.show .modal {
            transform: scale(1);
        }

        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .modal-title {
            font-size: 1.25rem;
            font-weight: 700;
        }

        .modal-close {
            background: none;
            border: none;
            font-size: 1.5rem;
            color: var(--text-secondary);
            cursor: pointer;
            line-height: 1;
        }

        .modal-close:hover {
            color: var(--text-primary);
        }

        .modal-footer {
            display: flex;
            justify-content: flex-end;
            gap: 1rem;
            margin-top: 1rem;
        }

        .btn-cancel {
            background-color: transparent;
            border: 1px solid var(--border-color);
            color: var(--text-secondary);
        }

        .btn-cancel:hover {
            background-color: rgba(255, 255, 255, 0.02);
            color: var(--text-primary);
            box-shadow: none;
        }

        /* ACTIONS PANEL INSIDE MODAL */
        .radio-actions {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 0.75rem;
        }

        .radio-btn {
            background-color: var(--bg-panel);
            border: 1px solid var(--border-color);
            border-radius: 10px;
            padding: 0.75rem;
            text-align: center;
            cursor: pointer;
            font-weight: 600;
            font-size: 0.875rem;
            transition: all 0.2s;
            user-select: none;
        }

        .radio-btn:hover {
            border-color: var(--text-secondary);
        }

        .radio-btn.active {
            background-color: rgba(139, 92, 246, 0.1);
            border-color: var(--accent-purple);
            color: var(--accent-purple);
        }

        /* ADMIN LOGIN */
        .login-overlay {
            position: fixed;
            inset: 0;
            background-color: rgba(9, 13, 22, 0.95);
            backdrop-filter: blur(8px);
            z-index: 200;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 1rem;
        }

        .login-overlay.hidden {
            display: none;
        }

        .login-card {
            background-color: var(--bg-surface);
            border: 1px solid var(--border-color);
            border-radius: 20px;
            padding: 2rem;
            width: 100%;
            max-width: 420px;
            display: flex;
            flex-direction: column;
            gap: 1.25rem;
        }

        .login-card h2 {
            font-size: 1.25rem;
            font-weight: 700;
        }

        .login-card p {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-top: -0.5rem;
        }

        /* RESPONSIVE DESIGN */
        @media (max-width: 900px) {
            .container {
                flex-direction: column;
                padding: 1rem;
            }
            .sidebar {
                width: 100%;
                flex-direction: row;
                overflow-x: auto;
                padding-bottom: 0.5rem;
            }
            .nav-btn {
                white-space: nowrap;
            }
            .nav-btn.active {
                border-left: none;
                border-bottom: 3px solid var(--accent-purple);
                border-radius: 12px 12px 0 0;
            }
        }
    </style>
</head>
<body>

    <header>
        <div class="logo">
            <span class="logo-icon">🎰</span>
            <span>Abdul Bot Panel</span>
        </div>
        <div class="bot-status" id="bot-status-badge">
            <span class="status-dot"></span>
            <span>Menghubungkan...</span>
        </div>
    </header>

    <div class="container">
        <!-- NAVIGATION SIDEBAR -->
        <aside class="sidebar">
            <button class="nav-btn active" onclick="switchTab('tab-overview', this)">
                📊 Ringkasan & Siaran
            </button>
            <button class="nav-btn" onclick="switchTab('tab-users', this); loadUsers()">
                👤 Kelola Saldo
            </button>
        </aside>

        <!-- VIEWPORTS -->
        <main class="main-content">
            <!-- TAB 1: OVERVIEW & BROADCAST -->
            <div id="tab-overview" class="tab-panel active">
                <div class="grid-stats">
                    <div class="card-stat">
                        <span class="stat-label">Grup Aktif</span>
                        <span class="stat-value" id="stat-groups">0</span>
                    </div>
                    <div class="card-stat">
                        <span class="stat-label">Game Poker</span>
                        <span class="stat-value" id="stat-poker">Nonaktif</span>
                    </div>
                    <div class="card-stat">
                        <span class="stat-label">Game Blackjack</span>
                        <span class="stat-value" id="stat-blackjack">Nonaktif</span>
                    </div>
                </div>

                <div class="panel">
                    <h2 class="panel-title">📢 Broadcast (Siaran) ke WhatsApp</h2>
                    <p class="panel-desc">Kirim pesan langsung ke satu atau semua grup terdaftar.</p>
                    
                    <form id="broadcast-form" onsubmit="sendBroadcast(event)">
                        <div class="form-group" style="margin-bottom: 1.5rem;">
                            <label for="broadcast-group">Pilih Grup Sasaran</label>
                            <select id="broadcast-group" required>
                                <option value="all">📢 Semua Grup Terdaftar</option>
                            </select>
                        </div>
                        <div class="form-group" style="margin-bottom: 1.5rem;">
                            <label for="broadcast-message">Isi Pesan Broadcast</label>
                            <textarea id="broadcast-message" placeholder="Tulis pesan siaran Anda di sini... Gunakan *tebal* untuk cetak tebal." required></textarea>
                        </div>
                        <button type="submit" class="btn" id="btn-broadcast">
                            🚀 Kirim Broadcast Sekarang
                        </button>
                    </form>
                </div>
            </div>

            <!-- TAB 2: USER BALANCE MANAGEMENT -->
            <div id="tab-users" class="tab-panel">
                <div class="panel">
                    <h2 class="panel-title">👥 Pengelolaan Saldo Chip Pemain</h2>
                    <p class="panel-desc">Cari pemain untuk menambah, mengurangi, atau menetapkan saldo secara persisten.</p>

                    <div style="display: flex; gap: 1rem; margin-bottom: 0.5rem;">
                        <input type="text" id="search-input" placeholder="Cari nama atau JID user..." oninput="filterUsers()">
                        <button class="btn" onclick="loadUsers()">🔄 Refresh</button>
                    </div>

                    <div class="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th>Nama Pemain</th>
                                    <th>WhatsApp JID</th>
                                    <th>Saldo Meja / Utama</th>
                                    <th>Aksi</th>
                                </tr>
                            </thead>
                            <tbody id="users-table-body">
                                <tr>
                                    <td colspan="4" style="text-align: center; color: var(--text-secondary);">Memuat data pemain...</td>
                                </tr>
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        </main>
    </div>

    <!-- BALANCE UPDATE MODAL -->
    <div class="modal-overlay" id="edit-modal-overlay">
        <div class="modal">
            <div class="modal-header">
                <h3 class="modal-title">✏️ Edit Saldo Pemain</h3>
                <button class="modal-close" onclick="closeModal()">&times;</button>
            </div>
            <div class="modal-desc" style="font-size: 0.875rem; color: var(--text-secondary); margin-top: -0.75rem;">
                Nama: <span id="modal-user-name" style="color: var(--text-primary); font-weight: 600;">-</span><br>
                JID: <span id="modal-user-jid" style="font-family: monospace; font-size: 0.75rem;">-</span>
            </div>

            <div class="form-group">
                <label>Aksi Perubahan</label>
                <div class="radio-actions">
                    <div class="radio-btn active" id="action-add" onclick="setAction('add')">➕ Tambah</div>
                    <div class="radio-btn" id="action-subtract" onclick="setAction('subtract')">➖ Kurang</div>
                    <div class="radio-btn" id="action-set" onclick="setAction('set')">⚙️ Set Baru</div>
                </div>
            </div>

            <div class="form-group">
                <label for="amount-input">Jumlah Chip</label>
                <input type="number" id="amount-input" min="0" placeholder="Masukkan jumlah chip..." required>
            </div>

            <div class="modal-footer">
                <button class="btn btn-cancel" onclick="closeModal()">Batal</button>
                <button class="btn" id="btn-save-balance" onclick="saveBalance()">Simpan Perubahan</button>
            </div>
        </div>
    </div>

    <!-- ADMIN TOKEN LOGIN -->
    <div class="login-overlay" id="login-overlay">
        <div class="login-card">
            <h2>🔐 Admin Panel</h2>
            <p>Masukkan token admin (<code>ADMIN_PANEL_TOKEN</code> dari file .env).</p>
            <div class="form-group">
                <label for="admin-token-input">Token</label>
                <input type="password" id="admin-token-input" placeholder="Bearer token..." autocomplete="current-password">
            </div>
            <button type="button" class="btn" onclick="submitAdminLogin()">Masuk</button>
        </div>
    </div>

    <!-- TOAST NOTIFICATION -->
    <div class="toast" id="toast-notif">
        <span id="toast-icon">ℹ️</span>
        <span id="toast-text">Notifikasi</span>
    </div>

    <script>
        let currentTab = 'tab-overview';
        let groups = [];
        let users = [];
        let selectedJID = '';
        let selectedAction = 'add'; // 'add', 'subtract', 'set'

        const ADMIN_TOKEN_KEY = 'adminPanelToken';

        function getAdminToken() {
            return sessionStorage.getItem(ADMIN_TOKEN_KEY) || '';
        }

        function showLoginOverlay() {
            document.getElementById('login-overlay').classList.remove('hidden');
        }

        function hideLoginOverlay() {
            document.getElementById('login-overlay').classList.add('hidden');
        }

        function submitAdminLogin() {
            const token = document.getElementById('admin-token-input').value.trim();
            if (!token) {
                showToast('❌ Token tidak boleh kosong.', 'error');
                return;
            }
            sessionStorage.setItem(ADMIN_TOKEN_KEY, token);
            hideLoginOverlay();
            fetchStatus();
            fetchGroups();
        }

        function adminFetch(url, options = {}) {
            const token = getAdminToken();
            if (!token) {
                showLoginOverlay();
                return Promise.reject(new Error('Not authenticated'));
            }
            options.headers = Object.assign({}, options.headers, {
                'Authorization': 'Bearer ' + token
            });
            return fetch(url, options).then(res => {
                if (res.status === 401) {
                    sessionStorage.removeItem(ADMIN_TOKEN_KEY);
                    showLoginOverlay();
                }
                return res;
            });
        }

        // INITIALIZATION
        window.addEventListener('DOMContentLoaded', () => {
            if (!getAdminToken()) {
                showLoginOverlay();
                return;
            }
            hideLoginOverlay();
            fetchStatus();
            fetchGroups();
        });

        // TAB NAVIGATION
        function switchTab(tabId, btn) {
            document.querySelectorAll('.tab-panel').forEach(panel => panel.classList.remove('active'));
            document.querySelectorAll('.nav-btn').forEach(nav => nav.classList.remove('active'));
            
            document.getElementById(tabId).classList.add('active');
            btn.classList.add('active');
            currentTab = tabId;
        }

        // FETCH STATUS INFO
        async function fetchStatus() {
            try {
                const res = await adminFetch('/admin/api/status');
                const data = await res.json();
                
                document.getElementById('stat-groups').textContent = data.groups_count;
                document.getElementById('stat-poker').textContent = data.poker_enabled ? 'Aktif' : 'Nonaktif';
                document.getElementById('stat-blackjack').textContent = data.blackjack_enabled ? 'Aktif' : 'Nonaktif';
                
                const badge = document.getElementById('bot-status-badge');
                badge.innerHTML = '<span class="status-dot"></span><span>Aktif & Online</span>';
            } catch (err) {
                console.error("Gagal memuat status:", err);
                const badge = document.getElementById('bot-status-badge');
                badge.innerHTML = '<span class="status-dot" style="background-color: var(--accent-rose); box-shadow: 0 0 8px var(--accent-rose)"></span><span style="color: var(--accent-rose)">Offline</span>';
                badge.style.background = 'rgba(239, 68, 68, 0.1)';
                badge.style.borderColor = 'rgba(239, 68, 68, 0.2)';
            }
        }

        // FETCH ALLOWED GROUPS
        async function fetchGroups() {
            try {
                const res = await adminFetch('/admin/api/groups');
                groups = await res.json();
                
                const select = document.getElementById('broadcast-group');
                // Simpan option pertama (all)
                select.innerHTML = '<option value="all">📢 Semua Grup Terdaftar</option>';
                
                groups.forEach(g => {
                    const opt = document.createElement('option');
                    opt.value = g.jid;
                    opt.textContent = "👥 " + g.name + " (" + g.jid + ")";
                    select.appendChild(opt);
                });
            } catch (err) {
                console.error("Gagal memuat grup:", err);
            }
        }

        // LOAD REGISTERED USERS
        async function loadUsers() {
            try {
                const tableBody = document.getElementById('users-table-body');
                tableBody.innerHTML = '<tr><td colspan="4" style="text-align: center; color: var(--text-secondary);">Memuat data pemain...</td></tr>';
                
                const query = document.getElementById('search-input').value;
                const res = await adminFetch("/admin/api/users?q=" + encodeURIComponent(query));
                users = await res.json();
                
                tableBody.innerHTML = '';
                
                if (users.length === 0) {
                    tableBody.innerHTML = '<tr><td colspan="4" style="text-align: center; color: var(--text-secondary);">Tidak ada data pemain terdaftar.</td></tr>';
                    return;
                }
                
                users.forEach(u => {
                    const row = document.createElement('tr');
                    
                    const nameCell = document.createElement('td');
                    nameCell.innerHTML = "<strong>" + escapeHTML(u.Name || 'Tanpa Nama') + "</strong>";
                    
                    const jidCell = document.createElement('td');
                    jidCell.innerHTML = '<span class="user-jid">' + escapeHTML(u.JID) + '</span>';
                    
                    const balCell = document.createElement('td');
                    balCell.innerHTML = '<span class="chip-balance">💰 ' + formatNumber(u.Balance) + ' chip</span>';
                    
                    const actionCell = document.createElement('td');
                    actionCell.innerHTML = '<button class="btn-edit" onclick="openEditModal(\'' + escapeHTML(u.JID) + '\', \'' + escapeHTML(u.Name || 'Tanpa Nama') + '\')">✏️ Edit Saldo</button>';
                    
                    row.appendChild(nameCell);
                    row.appendChild(jidCell);
                    row.appendChild(balCell);
                    row.appendChild(actionCell);
                    
                    tableBody.appendChild(row);
                });
            } catch (err) {
                console.error("Gagal memuat user:", err);
            }
        }

        // FILTER USERS ON INPUT (DEBOUNCED LATER OR INSTANT)
        let searchTimeout;
        function filterUsers() {
            clearTimeout(searchTimeout);
            searchTimeout = setTimeout(() => {
                loadUsers();
            }, 300);
        }

        // SEND BROADCAST
        async function sendBroadcast(e) {
            e.preventDefault();
            const btn = document.getElementById('btn-broadcast');
            const targetGroup = document.getElementById('broadcast-group').value;
            const message = document.getElementById('broadcast-message').value;
            
            btn.disabled = true;
            btn.textContent = '⏳ Mengirim siaran...';
            
            try {
                const res = await adminFetch('/admin/api/broadcast', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ group_jid: targetGroup, message: message })
                });
                const data = await res.json();
                
                if (res.ok && data.success) {
                    showToast('🎉 Broadcast berhasil dikirim!', 'success');
                    document.getElementById('broadcast-message').value = '';
                } else {
                    showToast('❌ Gagal: ' + (data.error || 'Terjadi kesalahan'), 'error');
                }
            } catch (err) {
                showToast('❌ Gagal menghubungi server', 'error');
                console.error(err);
            } finally {
                btn.disabled = false;
                btn.textContent = '🚀 Kirim Broadcast Sekarang';
            }
        }

        // OPEN EDIT MODAL
        function openEditModal(jid, name) {
            selectedJID = jid;
            document.getElementById('modal-user-name').textContent = name;
            document.getElementById('modal-user-jid').textContent = jid;
            document.getElementById('amount-input').value = '';
            setAction('add');
            
            document.getElementById('edit-modal-overlay').classList.add('show');
        }

        // CLOSE MODAL
        function closeModal() {
            document.getElementById('edit-modal-overlay').classList.remove('show');
        }

        // SET MODAL ACTION
        function setAction(action) {
            selectedAction = action;
            document.querySelectorAll('.radio-btn').forEach(btn => btn.classList.remove('active'));
            document.getElementById('action-' + action).classList.add('active');
        }

        // SAVE BALANCE UPDATE
        async function saveBalance() {
            const amount = parseInt(document.getElementById('amount-input').value);
            if (isNaN(amount) || amount < 0) {
                showToast('❌ Masukkan jumlah chip yang valid (>= 0).', 'error');
                return;
            }
            
            const btn = document.getElementById('btn-save-balance');
            btn.disabled = true;
            
            try {
                const res = await adminFetch('/admin/api/edit-balance', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ jid: selectedJID, amount: amount, action: selectedAction })
                });
                const data = await res.json();
                
                if (res.ok && data.success) {
                    showToast('✅ ' + data.message, 'success');
                    closeModal();
                    loadUsers();
                } else {
                    showToast('❌ Gagal: ' + (data.error || 'Terjadi kesalahan'), 'error');
                }
            } catch (err) {
                showToast('❌ Gagal menghubungi server', 'error');
                console.error(err);
            } finally {
                btn.disabled = false;
            }
        }

        // SHOW TOAST NOTIFICATION
        function showToast(text, type = 'info') {
            const toast = document.getElementById('toast-notif');
            const icon = document.getElementById('toast-icon');
            const textSpan = document.getElementById('toast-text');
            
            toast.className = 'toast show';
            if (type === 'success') {
                toast.classList.add('success');
                icon.textContent = '✅';
            } else if (type === 'error') {
                toast.classList.add('error');
                icon.textContent = '❌';
            } else {
                icon.textContent = 'ℹ️';
            }
            
            textSpan.textContent = text;
            
            setTimeout(() => {
                toast.classList.remove('show');
            }, 3000);
        }

        // HELPERS
        function escapeHTML(str) {
            return str.replace(/[&<>'"]/g, 
                tag => ({
                    '&': '&amp;',
                    '<': '&lt;',
                    '>': '&gt;',
                    "'": '&#39;',
                    '"': '&quot;'
                }[tag] || tag)
            );
        }

        function formatNumber(n) {
            return n.toString().replace(/\B(?=(\d{3})+(?!\n))/g, ".");
        }
    </script>
</body>
</html>`
