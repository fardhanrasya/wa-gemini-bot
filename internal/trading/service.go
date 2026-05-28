package trading

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"wa-gemini-bot/internal/economy"
)

// ─────────────────────────────────────────────────────────────
// ERRORS
// ─────────────────────────────────────────────────────────────

var (
	ErrInsufficientTradingBalance = errors.New("saldo trading tidak mencukupi")
	ErrInsufficientChips          = errors.New("saldo chip utama tidak mencukupi")
	ErrMinDepositNotMet           = errors.New("jumlah deposit di bawah minimum")
	ErrMinWithdrawNotMet          = errors.New("jumlah withdraw di bawah minimum")
	ErrNoActiveSession            = errors.New("tidak ada sesi trading aktif")
	ErrSessionExpired             = errors.New("sesi trading sudah kedaluwarsa")
	ErrPositionNotFound           = errors.New("posisi tidak ditemukan")
	ErrPositionAlreadyClosed      = errors.New("posisi sudah ditutup")
	ErrLeverageTooHigh            = errors.New("leverage melebihi batas rank kamu")
	ErrInvalidDirection           = errors.New("direction harus 'long' atau 'short'")
	ErrInvalidAmount              = errors.New("jumlah tidak valid")
	ErrInvalidPrice               = errors.New("harga tidak valid")
	ErrTradeCooldown              = errors.New("tunggu cooldown sebelum buka posisi lagi")
	ErrMaxTradesPerSession        = errors.New("batas transaksi sesi tercapai")
	ErrMaxOpenPositions           = errors.New("masih ada posisi aktif")
	ErrTradingSessionExpired      = errors.New("sesi login sudah kedaluwarsa")
	ErrInvalidLoginToken          = errors.New("token login tidak valid")
)

// ─────────────────────────────────────────────────────────────
// SERVICE
// ─────────────────────────────────────────────────────────────

// TradingService mengelola seluruh logika bisnis trading simulator.
// Saldo trading terpisah dari chip utama economy — pemain harus
// deposit chip ke akun trading dan bisa withdraw kembali kapan saja.
type TradingService struct {
	db            *sql.DB
	eco           *economy.EconomyService
	debugPassword string
	minDeposit    int
	minWithdraw   int

	// Cache sesi aktif di memory untuk performa.
	// Sesi aktif = chart session yang sedang dimainkan.
	mu             sync.RWMutex
	activeSessions map[string]*activeSession // jid -> session
}

// activeSession menyimpan state sesi yang sedang berjalan di memory.
type activeSession struct {
	ID          int
	Chart       *ChartSession
	StartTime   time.Time
	Positions   []Position
	Expired     bool
	LastTradeAt time.Time
}

// Position merepresentasikan satu posisi trading terbuka atau tertutup.
type Position struct {
	ID           int        `json:"id"`
	SessionID    int        `json:"session_id"`
	JID          string     `json:"-"`
	Direction    Direction  `json:"direction"`
	Leverage     int        `json:"leverage"`
	Size         int        `json:"size"` // margin dalam chip
	EntryPrice   float64    `json:"entry_price"`
	ExitPrice    *float64   `json:"exit_price"` // nil jika masih open
	StopLoss     *float64   `json:"stop_loss"`
	TakeProfit   *float64   `json:"take_profit"`
	TrailingStop *float64   `json:"trailing_stop"` // persentase trailing
	TrailingPeak float64    `json:"trailing_peak"` // peak P&L pct untuk trailing
	PnL          int        `json:"pnl"`
	Fee          int        `json:"fee"`
	Status       string     `json:"status"` // "open", "closed", "stopped_out", "take_profit", "trailing_stopped", "liquidated"
	OpenedAt     time.Time  `json:"opened_at"`
	ClosedAt     *time.Time `json:"closed_at"`
}

// TradingAccount berisi informasi akun trading pemain.
type TradingAccount struct {
	JID               string `json:"jid"`
	Balance           int    `json:"balance"`
	TotalDeposited    int    `json:"total_deposited"`
	TotalWithdrawn    int    `json:"total_withdrawn"`
	TotalProfit       int    `json:"total_profit"`
	TotalLoss         int    `json:"total_loss"`
	TotalSessions     int    `json:"total_sessions"`
	TotalTrades       int    `json:"total_trades"`
	TutorialCompleted bool   `json:"tutorial_completed"`
}

// TradingStats berisi statistik trading pemain untuk dashboard.
type TradingStats struct {
	WinRate       float64 `json:"win_rate"`
	TotalPnL      int     `json:"total_pnl"`
	BestTrade     int     `json:"best_trade"`
	WorstTrade    int     `json:"worst_trade"`
	AvgPnL        float64 `json:"avg_pnl"`
	TotalTrades   int     `json:"total_trades"`
	TotalSessions int     `json:"total_sessions"`
}

// OpenPositionRequest berisi parameter untuk membuka posisi baru.
type OpenPositionRequest struct {
	SessionID    int       `json:"session_id"`
	Direction    Direction `json:"direction"`
	Leverage     int       `json:"leverage"`
	Size         int       `json:"size"`
	StopLoss     *float64  `json:"stop_loss"`
	TakeProfit   *float64  `json:"take_profit"`
	TrailingStop *float64  `json:"trailing_stop"`
	EntryPrice   float64   `json:"entry_price"`
}

// TriggeredOrder merepresentasikan order stop/TP yang terpicu.
type TriggeredOrder struct {
	PositionID int     `json:"position_id"`
	Reason     string  `json:"reason"`
	ExitPrice  float64 `json:"exit_price"`
	PnL        int     `json:"pnl"`
}

// SessionSummary ringkasan satu sesi trading.
type SessionSummary struct {
	ID          int       `json:"id"`
	PatternType string    `json:"pattern_type"`
	PatternName string    `json:"pattern_name"`
	TotalPnL    int       `json:"total_pnl"`
	TradeCount  int       `json:"trade_count"`
	StartTime   time.Time `json:"start_time"`
	Status      string    `json:"status"`
}

// TradeRecord ringkasan satu posisi untuk riwayat.
type TradeRecord struct {
	ID         int       `json:"id"`
	Direction  Direction `json:"direction"`
	Leverage   int       `json:"leverage"`
	Size       int       `json:"size"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	PnL        int       `json:"pnl"`
	Status     string    `json:"status"`
	OpenedAt   time.Time `json:"opened_at"`
}

// ─────────────────────────────────────────────────────────────
// CONSTRUCTOR
// ─────────────────────────────────────────────────────────────

// NewTradingService membuat instance TradingService baru.
func NewTradingService(eco *economy.EconomyService, debugPassword string, minDeposit, minWithdraw int) *TradingService {
	return &TradingService{
		db:             eco.DB(),
		eco:            eco,
		debugPassword:  debugPassword,
		minDeposit:     minDeposit,
		minWithdraw:    minWithdraw,
		activeSessions: make(map[string]*activeSession),
	}
}

// ─────────────────────────────────────────────────────────────
// BALANCE MANAGEMENT
// ─────────────────────────────────────────────────────────────

// ensureTradingAccount memastikan akun trading sudah ada di database.
func (s *TradingService) ensureTradingAccount(jid string) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO trading_balances (jid, balance) VALUES (?, 0)`,
		jid,
	)
	return err
}

// GetTradingBalance mengembalikan informasi akun trading pemain.
func (s *TradingService) GetTradingBalance(jid string) (TradingAccount, error) {
	if err := s.ensureTradingAccount(jid); err != nil {
		return TradingAccount{}, err
	}

	var acc TradingAccount
	acc.JID = jid
	err := s.db.QueryRow(
		`SELECT balance, total_deposited, total_withdrawn, total_profit, total_loss, 
		        total_sessions, total_trades, tutorial_completed 
		 FROM trading_balances WHERE jid = ?`, jid,
	).Scan(&acc.Balance, &acc.TotalDeposited, &acc.TotalWithdrawn,
		&acc.TotalProfit, &acc.TotalLoss, &acc.TotalSessions,
		&acc.TotalTrades, &acc.TutorialCompleted)

	if err != nil {
		return TradingAccount{}, fmt.Errorf("gagal membaca akun trading: %w", err)
	}
	return acc, nil
}

// Deposit mentransfer chip dari saldo utama ke saldo trading (1:1).
func (s *TradingService) Deposit(jid string, amount int) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount < s.minDeposit {
		return fmt.Errorf("%w (minimum %d chip)", ErrMinDepositNotMet, s.minDeposit)
	}

	// 1. Potong dari saldo utama
	if err := s.eco.SubtractBalance(jid, amount, "trading_deposit", "deposit_to_trading"); err != nil {
		if errors.Is(err, economy.ErrInsufficientFunds) {
			return ErrInsufficientChips
		}
		return fmt.Errorf("gagal mengurangi saldo utama: %w", err)
	}

	// 2. Tambah ke saldo trading
	if err := s.ensureTradingAccount(jid); err != nil {
		// Refund jika gagal
		_ = s.eco.AddBalance(jid, amount, "trading_deposit_refund", "deposit_failed")
		return err
	}

	_, err := s.db.Exec(
		`UPDATE trading_balances SET balance = balance + ?, total_deposited = total_deposited + ? WHERE jid = ?`,
		amount, amount, jid,
	)
	if err != nil {
		// Refund jika gagal
		_ = s.eco.AddBalance(jid, amount, "trading_deposit_refund", "deposit_failed")
		return fmt.Errorf("gagal menambah saldo trading: %w", err)
	}

	log.Printf("[TRADING] 💰 DEPOSIT: %s deposit %d chip ke akun trading", jid, amount)
	return nil
}

// Withdraw mentransfer saldo trading kembali ke chip utama (1:1).
func (s *TradingService) Withdraw(jid string, amount int) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if amount < s.minWithdraw {
		return fmt.Errorf("%w (minimum %d chip)", ErrMinWithdrawNotMet, s.minWithdraw)
	}

	// 1. Cek dan kurangi saldo trading
	result, err := s.db.Exec(
		`UPDATE trading_balances SET balance = balance - ?, total_withdrawn = total_withdrawn + ? 
		 WHERE jid = ? AND balance >= ?`,
		amount, amount, jid, amount,
	)
	if err != nil {
		return fmt.Errorf("gagal mengurangi saldo trading: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrInsufficientTradingBalance
	}

	// 2. Tambah ke saldo utama
	if err := s.eco.AddBalance(jid, amount, "trading_withdraw", "withdraw_from_trading"); err != nil {
		// Rollback: kembalikan saldo trading
		_, _ = s.db.Exec(
			`UPDATE trading_balances SET balance = balance + ?, total_withdrawn = total_withdrawn - ? WHERE jid = ?`,
			amount, amount, jid,
		)
		return fmt.Errorf("gagal menambah saldo utama: %w", err)
	}

	log.Printf("[TRADING] 💸 WITHDRAW: %s withdraw %d chip dari akun trading", jid, amount)
	return nil
}

// ─────────────────────────────────────────────────────────────
// SESSION MANAGEMENT
// ─────────────────────────────────────────────────────────────

const sessionTimeout = 90 * time.Second // sesi auto-expire setelah 90 detik

const (
	tradingFeeRate             = 0.0015
	entrySpreadRate            = 0.0015
	tradeCooldown              = 6 * time.Second
	maxTradesPerSession        = 4
	maxOpenPositionsPerSession = 1
)

// StartSession membuat sesi trading baru dengan chart yang di-generate.
func (s *TradingService) StartSession(jid string) (*ChartSession, int, error) {
	// Pastikan akun trading ada
	if err := s.ensureTradingAccount(jid); err != nil {
		return nil, 0, err
	}

	// Cek apakah ada sesi aktif yang sedang berjalan dan belum timeout
	s.mu.Lock()
	if existing, ok := s.activeSessions[jid]; ok {
		if time.Since(existing.StartTime) < sessionTimeout {
			s.mu.Unlock()
			return nil, 0, fmt.Errorf("sesi trading aktif sedang berjalan. silakan selesaikan sesi ini terlebih dahulu")
		}
		// Sesi sebelumnya sudah habis waktu secara natural, tutup dulu secara internal
		s.mu.Unlock()
		s.endSessionInternal(jid, existing)
	} else {
		s.mu.Unlock()
	}

	// Generate chart
	chart := GenerateSession(jid)

	// Simpan ke database
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`INSERT INTO trading_sessions (jid, seed, pattern_type, pattern_name, pattern_resolved, noise_level, start_time, status) 
		 VALUES (?, ?, ?, ?, ?, ?, ?, 'active')`,
		jid, chart.Seed, chart.PatternType, chart.PatternName,
		chart.WillResolve, chart.NoiseLevel, now.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menyimpan sesi: %w", err)
	}

	sessionID64, _ := result.LastInsertId()
	sessionID := int(sessionID64)

	// Update counter
	_, _ = s.db.Exec(
		`UPDATE trading_balances SET total_sessions = total_sessions + 1 WHERE jid = ?`, jid,
	)

	// Cache di memory
	session := &activeSession{
		ID:        sessionID,
		Chart:     chart,
		StartTime: now,
		Positions: []Position{},
	}

	s.mu.Lock()
	s.activeSessions[jid] = session
	s.mu.Unlock()

	// Auto-expire setelah timeout
	go func() {
		time.Sleep(sessionTimeout)
		s.mu.RLock()
		current, ok := s.activeSessions[jid]
		s.mu.RUnlock()
		if ok && current.ID == sessionID {
			s.endSessionInternal(jid, current)
		}
	}()

	log.Printf("[TRADING] 📊 SESSION START: %s memulai sesi #%d (Pattern: %s, Resolve: %v, Noise: %.2f)",
		jid, sessionID, chart.PatternName, chart.WillResolve, chart.NoiseLevel)

	return chart, sessionID, nil
}

// GetResolutionData mengembalikan data chart resolution (fase setelah observasi).
// Hanya bisa dipanggil setelah fase observasi berakhir.
func (s *TradingService) GetResolutionData(jid string, sessionID int) ([]PricePoint, ChartIndicators, string, error) {
	s.mu.RLock()
	session, ok := s.activeSessions[jid]
	s.mu.RUnlock()

	if !ok || session.ID != sessionID {
		return nil, ChartIndicators{}, "", ErrNoActiveSession
	}

	// Hitung indikator untuk resolution
	allCandles := append(session.Chart.ObservationData, session.Chart.ResolutionData...)
	fullIndicators := calculateIndicators(allCandles)

	// Return hanya indikator fase resolution
	resIndicators := ChartIndicators{
		MA20:    fullIndicators.MA20[observationCandles:],
		RSI:     fullIndicators.RSI[observationCandles:],
		MACD:    fullIndicators.MACD[observationCandles:],
		MACDSig: fullIndicators.MACDSig[observationCandles:],
	}

	return session.Chart.ResolutionData, resIndicators, session.Chart.PatternName, nil
}

// GetActiveSession mengembalikan sesi aktif pemain (jika ada).
func (s *TradingService) GetActiveSession(jid string) (*activeSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.activeSessions[jid]
	return session, ok
}

// EndSession menutup sesi aktif secara manual/graceful.
func (s *TradingService) EndSession(jid string) error {
	s.mu.Lock()
	session, ok := s.activeSessions[jid]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("tidak ada sesi aktif")
	}

	s.endSessionInternal(jid, session)
	return nil
}

// endSessionInternal menutup sesi aktif: close semua posisi terbuka, hitung P&L.
func (s *TradingService) endSessionInternal(jid string, session *activeSession) {
	s.mu.Lock()
	if current, ok := s.activeSessions[jid]; !ok || current.ID != session.ID {
		s.mu.Unlock()
		return
	}
	delete(s.activeSessions, jid)
	s.mu.Unlock()

	totalPnL := 0

	// Auto-close semua posisi terbuka
	for i := range session.Positions {
		pos := &session.Positions[i]
		if pos.Status == "open" {
			// Close at last known price
			lastCandle := session.Chart.ResolutionData[len(session.Chart.ResolutionData)-1]
			exitPrice := lastCandle.Close
			pnl := calculatePnL(*pos, exitPrice)

			pos.ExitPrice = &exitPrice
			pos.PnL = pnl
			pos.Status = "closed"
			now := time.Now()
			pos.ClosedAt = &now

			// Update di database
			_, _ = s.db.Exec(
				`UPDATE trading_trades SET exit_price = ?, pnl = ?, status = ?, closed_at = ? WHERE id = ?`,
				exitPrice, pnl, "closed", now.Format("2006-01-02 15:04:05"), pos.ID,
			)

			s.finalizePosition(jid, pos)
		}
		totalPnL += pos.PnL
	}

	// Update sesi di database
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, _ = s.db.Exec(
		`UPDATE trading_sessions SET end_time = ?, total_pnl = ?, status = 'completed' WHERE id = ?`,
		now, totalPnL, session.ID,
	)

	log.Printf("[TRADING] 📊 SESSION END: %s sesi #%d selesai (Total P&L: %d)", jid, session.ID, totalPnL)
}

// ─────────────────────────────────────────────────────────────
// POSITION MANAGEMENT
// ─────────────────────────────────────────────────────────────

// OpenPosition membuka posisi LONG atau SHORT dalam sesi aktif.
func (s *TradingService) OpenPosition(jid string, req OpenPositionRequest) (*Position, error) {
	// Validasi direction
	if req.Direction != DirBullish && req.Direction != DirBearish {
		return nil, ErrInvalidDirection
	}
	if req.Size <= 0 {
		return nil, ErrInvalidAmount
	}
	if !isValidPrice(req.EntryPrice) {
		return nil, ErrInvalidPrice
	}
	if req.Leverage < 1 {
		req.Leverage = 1
	}

	// Validasi leverage vs rank
	chipBalance, err := s.eco.GetBalance(jid)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca saldo chip: %w", err)
	}
	maxLev := MaxLeverageForBalance(chipBalance)
	if req.Leverage > maxLev {
		return nil, fmt.Errorf("%w (max: %dx untuk rank kamu)", ErrLeverageTooHigh, maxLev)
	}

	// Cek sesi aktif
	s.mu.Lock()
	session, ok := s.activeSessions[jid]
	if !ok {
		s.mu.Unlock()
		return nil, ErrNoActiveSession
	}
	if req.SessionID != 0 && req.SessionID != session.ID {
		s.mu.Unlock()
		return nil, ErrNoActiveSession
	}
	if len(session.Positions) >= maxTradesPerSession {
		s.mu.Unlock()
		return nil, fmt.Errorf("%w (maksimal %d transaksi per sesi)", ErrMaxTradesPerSession, maxTradesPerSession)
	}
	if !session.LastTradeAt.IsZero() {
		remaining := tradeCooldown - time.Since(session.LastTradeAt)
		if remaining > 0 {
			s.mu.Unlock()
			return nil, fmt.Errorf("%w (%d detik)", ErrTradeCooldown, int(math.Ceil(remaining.Seconds())))
		}
	}

	// Cek saldo trading mencukupi
	acc, err := s.GetTradingBalance(jid)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}

	// Hitung margin yang sudah terkunci di posisi terbuka
	lockedMargin := 0
	openPositions := 0
	for _, pos := range session.Positions {
		if pos.Status == "open" {
			lockedMargin += pos.Size
			openPositions++
		}
	}
	if openPositions >= maxOpenPositionsPerSession {
		s.mu.Unlock()
		return nil, fmt.Errorf("%w. tutup posisi aktif sebelum buka posisi baru", ErrMaxOpenPositions)
	}

	availableBalance := acc.Balance - lockedMargin
	if req.Size > availableBalance {
		s.mu.Unlock()
		return nil, fmt.Errorf("%w (tersedia: %d chip, butuh: %d chip)", ErrInsufficientTradingBalance, availableBalance, req.Size)
	}

	entryPrice := applyEntrySpread(req.Direction, req.EntryPrice)

	// Buat posisi baru
	now := time.Now()
	pos := Position{
		SessionID:    session.ID,
		JID:          jid,
		Direction:    req.Direction,
		Leverage:     req.Leverage,
		Size:         req.Size,
		EntryPrice:   entryPrice,
		StopLoss:     req.StopLoss,
		TakeProfit:   req.TakeProfit,
		TrailingStop: req.TrailingStop,
		Fee:          calculateTradingFee(req.Size, req.Leverage),
		Status:       "open",
		OpenedAt:     now,
	}

	// Simpan ke database
	result, errDB := s.db.Exec(
		`INSERT INTO trading_trades (session_id, jid, direction, leverage, size, entry_price, stop_loss, take_profit, trailing_stop, status, opened_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'open', ?)`,
		session.ID, jid, string(req.Direction), req.Leverage, req.Size,
		entryPrice, req.StopLoss, req.TakeProfit, req.TrailingStop,
		now.Format("2006-01-02 15:04:05"),
	)
	if errDB != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("gagal menyimpan posisi: %w", errDB)
	}

	id64, _ := result.LastInsertId()
	pos.ID = int(id64)

	session.Positions = append(session.Positions, pos)
	session.LastTradeAt = now
	s.mu.Unlock()

	// Update trade counter
	_, _ = s.db.Exec(`UPDATE trading_balances SET total_trades = total_trades + 1 WHERE jid = ?`, jid)

	dirLabel := "LONG 📈"
	if req.Direction == DirBearish {
		dirLabel = "SHORT 📉"
	}
	log.Printf("[TRADING] 🔓 OPEN: %s %s %dx | Size: %d | Entry: %.2f | Fee: %d | SL: %v | TP: %v",
		jid, dirLabel, req.Leverage, req.Size, entryPrice, pos.Fee, req.StopLoss, req.TakeProfit)

	return &pos, nil
}

// ClosePosition menutup posisi secara manual pada harga yang diberikan.
func (s *TradingService) ClosePosition(jid string, positionID int, exitPrice float64) (*Position, error) {
	if !isValidPrice(exitPrice) {
		return nil, ErrInvalidPrice
	}

	s.mu.Lock()
	session, ok := s.activeSessions[jid]
	if !ok {
		s.mu.Unlock()
		return nil, ErrNoActiveSession
	}

	var pos *Position
	for i := range session.Positions {
		if session.Positions[i].ID == positionID {
			pos = &session.Positions[i]
			break
		}
	}
	s.mu.Unlock()

	if pos == nil {
		return nil, ErrPositionNotFound
	}
	if pos.Status != "open" {
		return nil, ErrPositionAlreadyClosed
	}

	return s.closePositionInternal(jid, pos, exitPrice, "closed")
}

// closePositionInternal menutup posisi dengan status dan harga yang diberikan.
func (s *TradingService) closePositionInternal(jid string, pos *Position, exitPrice float64, status string) (*Position, error) {
	pnl := calculatePnL(*pos, exitPrice)

	pos.ExitPrice = &exitPrice
	pos.PnL = pnl
	pos.Status = status
	now := time.Now()
	pos.ClosedAt = &now

	// Update di database
	_, err := s.db.Exec(
		`UPDATE trading_trades SET exit_price = ?, pnl = ?, status = ?, closed_at = ? WHERE id = ?`,
		exitPrice, pnl, status, now.Format("2006-01-02 15:04:05"), pos.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal menutup posisi: %w", err)
	}

	s.finalizePosition(jid, pos)
	s.markTradeAction(jid, now)

	log.Printf("[TRADING] 🔒 CLOSE: %s posisi #%d | Exit: %.2f | P&L: %d | Status: %s",
		jid, pos.ID, exitPrice, pnl, status)

	return pos, nil
}

// finalizePosition mengupdate saldo trading berdasarkan P&L posisi yang ditutup.
func (s *TradingService) finalizePosition(jid string, pos *Position) {
	if pos.PnL > 0 {
		_, _ = s.db.Exec(
			`UPDATE trading_balances SET balance = balance + ?, total_profit = total_profit + ? WHERE jid = ?`,
			pos.PnL, pos.PnL, jid,
		)
	} else if pos.PnL < 0 {
		loss := int(math.Abs(float64(pos.PnL)))
		// Pastikan saldo tidak negatif
		_, _ = s.db.Exec(
			`UPDATE trading_balances SET balance = MAX(0, balance - ?), total_loss = total_loss + ? WHERE jid = ?`,
			loss, loss, jid,
		)
	}
}

// SetStopLoss mengatur stop loss pada posisi terbuka.
func (s *TradingService) SetStopLoss(jid string, positionID int, price float64) error {
	return s.updatePositionField(jid, positionID, "stop_loss", price)
}

// SetTakeProfit mengatur take profit pada posisi terbuka.
func (s *TradingService) SetTakeProfit(jid string, positionID int, price float64) error {
	return s.updatePositionField(jid, positionID, "take_profit", price)
}

// SetTrailingStop mengatur trailing stop (persentase) pada posisi terbuka.
func (s *TradingService) SetTrailingStop(jid string, positionID int, pct float64) error {
	return s.updatePositionField(jid, positionID, "trailing_stop", pct)
}

func (s *TradingService) updatePositionField(jid string, posID int, field string, value float64) error {
	s.mu.Lock()
	session, ok := s.activeSessions[jid]
	if !ok {
		s.mu.Unlock()
		return ErrNoActiveSession
	}

	for i := range session.Positions {
		if session.Positions[i].ID == posID && session.Positions[i].Status == "open" {
			switch field {
			case "stop_loss":
				session.Positions[i].StopLoss = &value
			case "take_profit":
				session.Positions[i].TakeProfit = &value
			case "trailing_stop":
				session.Positions[i].TrailingStop = &value
			}
			s.mu.Unlock()

			_, err := s.db.Exec(
				fmt.Sprintf(`UPDATE trading_trades SET %s = ? WHERE id = ?`, field),
				value, posID,
			)
			return err
		}
	}
	s.mu.Unlock()
	return ErrPositionNotFound
}

// CheckStopOrders memeriksa apakah stop loss, take profit, atau trailing stop
// terpicu pada harga saat ini. Dipanggil oleh client setiap kali candle baru muncul.
func (s *TradingService) CheckStopOrders(jid string, currentPrice float64) []TriggeredOrder {
	if !isValidPrice(currentPrice) {
		return nil
	}

	s.mu.Lock()
	session, ok := s.activeSessions[jid]
	if !ok {
		s.mu.Unlock()
		return nil
	}

	var triggered []TriggeredOrder

	for i := range session.Positions {
		pos := &session.Positions[i]
		if pos.Status != "open" {
			continue
		}

		reason := checkStopConditions(pos, currentPrice)
		if reason == "" {
			continue
		}

		// Tutup posisi
		s.mu.Unlock()
		closedPos, err := s.closePositionInternal(jid, pos, currentPrice, reason)
		s.mu.Lock()

		if err == nil {
			triggered = append(triggered, TriggeredOrder{
				PositionID: closedPos.ID,
				Reason:     reason,
				ExitPrice:  currentPrice,
				PnL:        closedPos.PnL,
			})
		}
	}

	s.mu.Unlock()
	return triggered
}

// checkStopConditions memeriksa semua kondisi stop pada posisi.
func checkStopConditions(pos *Position, currentPrice float64) string {
	// 1. Liquidation: loss >= 90% margin
	pnl := calculatePnL(*pos, currentPrice)
	liquidationThreshold := -float64(pos.Size) * 0.90
	if float64(pnl) <= liquidationThreshold {
		return "liquidated"
	}

	// 2. Stop Loss
	if pos.StopLoss != nil {
		if pos.Direction == DirBullish && currentPrice <= *pos.StopLoss {
			return "stopped_out"
		}
		if pos.Direction == DirBearish && currentPrice >= *pos.StopLoss {
			return "stopped_out"
		}
	}

	// 3. Take Profit
	if pos.TakeProfit != nil {
		if pos.Direction == DirBullish && currentPrice >= *pos.TakeProfit {
			return "take_profit"
		}
		if pos.Direction == DirBearish && currentPrice <= *pos.TakeProfit {
			return "take_profit"
		}
	}

	// 4. Trailing Stop
	if pos.TrailingStop != nil {
		currentPnLPct := calculatePnLPercent(*pos, currentPrice)
		if currentPnLPct > pos.TrailingPeak {
			pos.TrailingPeak = currentPnLPct
		}
		if pos.TrailingPeak > 0 {
			drawdown := pos.TrailingPeak - currentPnLPct
			if drawdown >= *pos.TrailingStop {
				return "trailing_stopped"
			}
		}
	}

	return ""
}

// ─────────────────────────────────────────────────────────────
// P&L CALCULATION
// ─────────────────────────────────────────────────────────────

// calculatePnL menghitung profit/loss dalam chip.
// Formula sama persis dengan real trading:
//
//	PnL = priceChange% × leverage × margin
func calculatePnL(pos Position, currentPrice float64) int {
	if !isValidPrice(pos.EntryPrice) || !isValidPrice(currentPrice) {
		return 0
	}

	var priceChange float64
	if pos.Direction == DirBullish {
		priceChange = (currentPrice - pos.EntryPrice) / pos.EntryPrice
	} else {
		priceChange = (pos.EntryPrice - currentPrice) / pos.EntryPrice
	}

	pnl := priceChange * float64(pos.Leverage) * float64(pos.Size)
	return int(math.Round(pnl)) - calculatePositionFee(pos)
}

// calculatePnLPercent menghitung P&L dalam persentase (untuk trailing stop).
func calculatePnLPercent(pos Position, currentPrice float64) float64 {
	if !isValidPrice(pos.EntryPrice) || !isValidPrice(currentPrice) {
		return 0
	}

	var priceChange float64
	if pos.Direction == DirBullish {
		priceChange = (currentPrice - pos.EntryPrice) / pos.EntryPrice
	} else {
		priceChange = (pos.EntryPrice - currentPrice) / pos.EntryPrice
	}

	return priceChange * float64(pos.Leverage) * 100
}

func isValidPrice(price float64) bool {
	return price > 0 && !math.IsNaN(price) && !math.IsInf(price, 0)
}

func applyEntrySpread(direction Direction, marketPrice float64) float64 {
	if direction == DirBullish {
		return marketPrice * (1 + entrySpreadRate)
	}
	return marketPrice * (1 - entrySpreadRate)
}

func calculateTradingFee(size, leverage int) int {
	if size <= 0 || leverage <= 0 {
		return 0
	}
	return int(math.Ceil(float64(size*leverage) * tradingFeeRate))
}

func calculatePositionFee(pos Position) int {
	if pos.Fee > 0 {
		return pos.Fee
	}
	return calculateTradingFee(pos.Size, pos.Leverage)
}

func (s *TradingService) markTradeAction(jid string, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, ok := s.activeSessions[jid]; ok {
		session.LastTradeAt = at
	}
}

// ─────────────────────────────────────────────────────────────
// HISTORY & STATS
// ─────────────────────────────────────────────────────────────

// GetTradeHistory mengembalikan riwayat trade terakhir.
func (s *TradingService) GetTradeHistory(jid string, limit int) ([]TradeRecord, error) {
	rows, err := s.db.Query(
		`SELECT id, direction, leverage, size, entry_price, COALESCE(exit_price, 0), pnl, status, opened_at 
		 FROM trading_trades WHERE jid = ? AND status != 'open' 
		 ORDER BY id DESC LIMIT ?`,
		jid, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []TradeRecord
	for rows.Next() {
		var r TradeRecord
		var openedStr string
		err := rows.Scan(&r.ID, &r.Direction, &r.Leverage, &r.Size,
			&r.EntryPrice, &r.ExitPrice, &r.PnL, &r.Status, &openedStr)
		if err != nil {
			return nil, err
		}
		r.OpenedAt, _ = time.Parse("2006-01-02 15:04:05", openedStr)
		records = append(records, r)
	}
	return records, nil
}

// GetSessionHistory mengembalikan riwayat sesi terakhir.
func (s *TradingService) GetSessionHistory(jid string, limit int) ([]SessionSummary, error) {
	rows, err := s.db.Query(
		`SELECT s.id, s.pattern_type, s.pattern_name, s.total_pnl, s.status, s.start_time,
		        (SELECT COUNT(*) FROM trading_trades t WHERE t.session_id = s.id) as trade_count
		 FROM trading_sessions s WHERE s.jid = ? 
		 ORDER BY s.id DESC LIMIT ?`,
		jid, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionSummary
	for rows.Next() {
		var ss SessionSummary
		var startStr string
		err := rows.Scan(&ss.ID, &ss.PatternType, &ss.PatternName,
			&ss.TotalPnL, &ss.Status, &startStr, &ss.TradeCount)
		if err != nil {
			return nil, err
		}
		ss.StartTime, _ = time.Parse("2006-01-02 15:04:05", startStr)
		sessions = append(sessions, ss)
	}
	return sessions, nil
}

// GetTradingStats mengembalikan statistik trading pemain.
func (s *TradingService) GetTradingStats(jid string) (TradingStats, error) {
	var stats TradingStats

	// Win rate
	var wins, total int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM trading_trades WHERE jid = ? AND status != 'open'`, jid,
	).Scan(&total)
	if err != nil {
		return stats, err
	}

	_ = s.db.QueryRow(
		`SELECT COUNT(*) FROM trading_trades WHERE jid = ? AND status != 'open' AND pnl > 0`, jid,
	).Scan(&wins)

	if total > 0 {
		stats.WinRate = float64(wins) / float64(total) * 100
	}
	stats.TotalTrades = total

	// Total P&L, best, worst, average
	_ = s.db.QueryRow(
		`SELECT COALESCE(SUM(pnl), 0), COALESCE(MAX(pnl), 0), COALESCE(MIN(pnl), 0), COALESCE(AVG(pnl), 0)
		 FROM trading_trades WHERE jid = ? AND status != 'open'`, jid,
	).Scan(&stats.TotalPnL, &stats.BestTrade, &stats.WorstTrade, &stats.AvgPnL)

	// Total sessions
	_ = s.db.QueryRow(
		`SELECT COUNT(*) FROM trading_sessions WHERE jid = ?`, jid,
	).Scan(&stats.TotalSessions)

	return stats, nil
}

// ─────────────────────────────────────────────────────────────
// TUTORIAL
// ─────────────────────────────────────────────────────────────

// GetTutorialProgress mengembalikan status tutorial pemain.
func (s *TradingService) GetTutorialProgress(jid string) (*TutorialProgress, error) {
	if err := s.ensureTradingAccount(jid); err != nil {
		return nil, err
	}

	var completed int
	var practiceDone bool
	err := s.db.QueryRow(
		`SELECT COALESCE(tutorial_step, 0), tutorial_completed FROM trading_balances WHERE jid = ?`, jid,
	).Scan(&completed, &practiceDone)

	if err != nil {
		// Kolom mungkin belum ada, return default
		return &TutorialProgress{JID: jid}, nil
	}

	return &TutorialProgress{
		JID:            jid,
		CompletedSteps: completed,
		PracticeDone:   practiceDone,
	}, nil
}

// CompleteTutorialStep menandai satu step tutorial sebagai selesai.
func (s *TradingService) CompleteTutorialStep(jid string, step int) error {
	_, err := s.db.Exec(
		`UPDATE trading_balances SET tutorial_step = MAX(COALESCE(tutorial_step, 0), ?) WHERE jid = ?`,
		step, jid,
	)
	return err
}

// CompletePractice menandai sesi latihan sebagai selesai.
func (s *TradingService) CompletePractice(jid string) error {
	_, err := s.db.Exec(
		`UPDATE trading_balances SET tutorial_completed = 1, tutorial_step = ? WHERE jid = ?`,
		len(TutorialSteps), jid,
	)
	return err
}

// StartPracticeSession membuat sesi latihan tutorial (noise minimal, pattern mudah).
func (s *TradingService) StartPracticeSession(jid string) (*ChartSession, error) {
	chart := GeneratePracticeSession(jid)
	return chart, nil
}

// ─────────────────────────────────────────────────────────────
// DEBUG
// ─────────────────────────────────────────────────────────────

// ValidateDebugPassword memeriksa apakah password debug benar.
func (s *TradingService) ValidateDebugPassword(password string) bool {
	return s.debugPassword != "" && password == s.debugPassword
}

// DebugSetBalance langsung set saldo trading (untuk testing).
func (s *TradingService) DebugSetBalance(jid string, amount int) error {
	if err := s.ensureTradingAccount(jid); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE trading_balances SET balance = ? WHERE jid = ?`, amount, jid)
	return err
}

// DebugGetPatternList mengembalikan daftar semua pattern yang tersedia.
func (s *TradingService) DebugGetPatternList() []map[string]string {
	var list []map[string]string
	for _, p := range allPatterns {
		list = append(list, map[string]string{
			"type":       p.Type(),
			"name":       p.Name(),
			"direction":  string(p.ExpectedDirection()),
			"difficulty": p.Difficulty(),
		})
	}
	return list
}

// ─────────────────────────────────────────────────────────────
// AUTHENTICATION (Reuse pattern dari Mining)
// ─────────────────────────────────────────────────────────────

func generateSecureToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateLoginToken membuat token satu-kali login (berlaku 1 jam).
func (s *TradingService) CreateLoginToken(jid string) (string, error) {
	token := generateSecureToken()
	expiresAt := time.Now().UTC().Add(1 * time.Hour).Format("2006-01-02 15:04:05")

	_, _ = s.db.Exec("DELETE FROM web_login_tokens WHERE jid = ? AND token LIKE 'trd_%'", jid)

	prefixedToken := "trd_" + token
	_, err := s.db.Exec(
		"INSERT INTO web_login_tokens (token, jid, expires_at) VALUES (?, ?, ?)",
		prefixedToken, jid, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan token login trading: %w", err)
	}

	return prefixedToken, nil
}

// VerifyLoginToken memverifikasi token login satu-kali.
func (s *TradingService) VerifyLoginToken(token string) (string, error) {
	var jid, expiresStr string
	err := s.db.QueryRow(
		"SELECT jid, expires_at FROM web_login_tokens WHERE token = ?", token,
	).Scan(&jid, &expiresStr)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidLoginToken
		}
		return "", err
	}

	expiresAt, errParse := time.Parse("2006-01-02 15:04:05", expiresStr)
	if errParse != nil {
		expiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	}

	if time.Now().After(expiresAt) {
		_, _ = s.db.Exec("DELETE FROM web_login_tokens WHERE token = ?", token)
		return "", ErrInvalidLoginToken
	}

	_, _ = s.db.Exec("DELETE FROM web_login_tokens WHERE token = ?", token)
	return jid, nil
}

// CreateSession membuat cookie session ID (berlaku 7 hari).
func (s *TradingService) CreateSession(jid string) (string, error) {
	sessionID := generateSecureToken()
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour).Format("2006-01-02 15:04:05")

	_, err := s.db.Exec(
		"INSERT INTO web_sessions (session_id, jid, expires_at) VALUES (?, ?, ?)",
		"trd_"+sessionID, jid, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("gagal menyimpan sesi trading: %w", err)
	}
	return "trd_" + sessionID, nil
}

// VerifySession memvalidasi session ID dari Cookie.
func (s *TradingService) VerifySession(sessionID string) (string, error) {
	var jid, expiresStr string
	err := s.db.QueryRow(
		"SELECT jid, expires_at FROM web_sessions WHERE session_id = ?", sessionID,
	).Scan(&jid, &expiresStr)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrTradingSessionExpired
		}
		return "", err
	}

	expiresAt, errParse := time.Parse("2006-01-02 15:04:05", expiresStr)
	if errParse != nil {
		expiresAt, _ = time.Parse(time.RFC3339, expiresStr)
	}

	if time.Now().After(expiresAt) {
		_, _ = s.db.Exec("DELETE FROM web_sessions WHERE session_id = ?", sessionID)
		return "", ErrTradingSessionExpired
	}

	return jid, nil
}

// LeaderboardEntry merepresentasikan satu baris dalam papan peringkat trading.
type LeaderboardEntry struct {
	Rank          int    `json:"rank"`
	Username      string `json:"username"`
	Balance       int    `json:"balance"`
	TotalProfit   int    `json:"total_profit"`
	TotalLoss     int    `json:"total_loss"`
	TotalTrades   int    `json:"total_trades"`
	TotalSessions int    `json:"total_sessions"`
}

// GetLeaderboard mengambil peringkat trader teratas berdasarkan saldo trading.
func (s *TradingService) GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	rows, err := s.db.Query(`
		SELECT jid, balance, total_profit, total_loss, total_trades, total_sessions 
		FROM trading_balances 
		ORDER BY balance DESC, total_profit DESC 
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat data leaderboard: %w", err)
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	rank := 1
	for rows.Next() {
		var entry LeaderboardEntry
		var jid string
		err := rows.Scan(
			&jid, &entry.Balance, &entry.TotalProfit,
			&entry.TotalLoss, &entry.TotalTrades, &entry.TotalSessions,
		)
		if err != nil {
			return nil, err
		}

		entry.Rank = rank
		entry.Username = strings.Split(jid, "@")[0] // Ambil nama depan JID
		leaderboard = append(leaderboard, entry)
		rank++
	}
	return leaderboard, nil
}

// DestroySession menghapus sesi aktif.
func (s *TradingService) DestroySession(sessionID string) {
	_, _ = s.db.Exec("DELETE FROM web_sessions WHERE session_id = ?", sessionID)
}
