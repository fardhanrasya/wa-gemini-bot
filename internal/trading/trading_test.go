package trading

import (
	"database/sql"
	"errors"
	"math"
	"os"
	"testing"

	"wa-gemini-bot/internal/economy"
)

func setupTestDB(t *testing.T) (*sql.DB, *economy.EconomyService) {
	dbPath := "test_trading.db"
	_ = os.Remove(dbPath)

	eco, err := economy.NewEconomyService(dbPath)
	if err != nil {
		t.Fatalf("Gagal inisialisasi economy: %v", err)
	}

	// Trigger migrations for trading in economy.go
	db := eco.DB()

	return db, eco
}

func teardownTestDB(t *testing.T, eco *economy.EconomyService) {
	_ = eco.Close()
	_ = os.Remove("test_trading.db")
}

func TestTradingServiceLifecycle(t *testing.T) {
	_, eco := setupTestDB(t)
	defer teardownTestDB(t, eco)

	// Create service
	debugPwd := "supersecret"
	svc := NewTradingService(eco, debugPwd, 500, 100)

	jid := "628123456789@s.whatsapp.net"

	// 1. Check account creation
	acc, err := svc.GetTradingBalance(jid)
	if err != nil {
		t.Fatalf("GetTradingBalance error: %v", err)
	}
	if acc.Balance != 0 {
		t.Errorf("Expected balance 0, got %d", acc.Balance)
	}

	// 2. Deposit check
	// Give main wallet balance first
	err = eco.AddBalance(jid, 5000, "test", "setup")
	if err != nil {
		t.Fatalf("AddBalance error: %v", err)
	}

	// Deposit under minimum
	err = svc.Deposit(jid, 100)
	if err == nil {
		t.Error("Expected error depositing below minimum 500, got nil")
	}

	// Deposit valid amount
	err = svc.Deposit(jid, 1000)
	if err != nil {
		t.Fatalf("Deposit error: %v", err)
	}

	acc, _ = svc.GetTradingBalance(jid)
	if acc.Balance != 1000 {
		t.Errorf("Expected balance 1000, got %d", acc.Balance)
	}

	mainBal, _ := eco.GetBalance(jid)
	if mainBal != 9000 {
		t.Errorf("Expected main wallet balance 9000, got %d", mainBal)
	}

	// 3. Start Sesi
	chart, sessionID, err := svc.StartSession(jid)
	if err != nil {
		t.Fatalf("StartSession error: %v", err)
	}
	if sessionID == 0 {
		t.Error("Expected valid session ID, got 0")
	}
	if len(chart.ObservationData) != observationCandles {
		t.Errorf("Expected %d observation candles, got %d", observationCandles, len(chart.ObservationData))
	}

	// 4. Open Position Validation
	// Buy position LONG with size 100, leverage 2x
	req := OpenPositionRequest{
		SessionID:  sessionID,
		Direction:  DirBullish,
		Leverage:   2,
		Size:       100,
		EntryPrice: chart.ObservationData[len(chart.ObservationData)-1].Close,
	}

	pos, err := svc.OpenPosition(jid, req)
	if err != nil {
		t.Fatalf("OpenPosition error: %v", err)
	}
	if pos.Status != "open" {
		t.Errorf("Expected position status 'open', got '%s'", pos.Status)
	}
	if pos.Size != 100 {
		t.Errorf("Expected size 100, got %d", pos.Size)
	}

	// 5. Check Stop orders triggering
	// Set Stop Loss at entry_price * 0.95
	slPrice := pos.EntryPrice * 0.95
	err = svc.SetStopLoss(jid, pos.ID, slPrice)
	if err != nil {
		t.Fatalf("SetStopLoss error: %v", err)
	}

	// Verify stop price was hit (currentPrice <= slPrice for Long)
	triggered := svc.CheckStopOrders(jid, slPrice)
	if len(triggered) != 1 {
		t.Fatalf("Expected 1 order triggered, got %d", len(triggered))
	}
	if triggered[0].PositionID != pos.ID {
		t.Errorf("Expected triggered position %d, got %d", pos.ID, triggered[0].PositionID)
	}
	if triggered[0].Reason != "stopped_out" {
		t.Errorf("Expected reason 'stopped_out', got '%s'", triggered[0].Reason)
	}

	// Check balance after loss
	acc, _ = svc.GetTradingBalance(jid)
	expectedLoss := int(math.Round(0.05*2*100)) + calculateTradingFee(pos.Size, pos.Leverage)
	expectedBalance := 1000 - expectedLoss
	if acc.Balance != expectedBalance {
		t.Errorf("Expected balance %d after loss, got %d", expectedBalance, acc.Balance)
	}

	// 6. Withdraw check
	// Withdraw under minimum
	err = svc.Withdraw(jid, 50)
	if err == nil {
		t.Error("Expected error withdrawing below minimum 100, got nil")
	}

	// Valid withdraw
	err = svc.Withdraw(jid, 500)
	if err != nil {
		t.Fatalf("Withdraw error: %v", err)
	}

	acc, _ = svc.GetTradingBalance(jid)
	if acc.Balance != expectedBalance-500 {
		t.Errorf("Expected balance %d, got %d", expectedBalance-500, acc.Balance)
	}

	mainBal, _ = eco.GetBalance(jid)
	if mainBal != 9500 {
		t.Errorf("Expected main wallet balance 9500, got %d", mainBal)
	}

	// 7. Debug operations
	if !svc.ValidateDebugPassword(debugPwd) {
		t.Error("Debug password validation failed")
	}

	err = svc.DebugSetBalance(jid, 10000)
	if err != nil {
		t.Fatalf("DebugSetBalance error: %v", err)
	}
	acc, _ = svc.GetTradingBalance(jid)
	if acc.Balance != 10000 {
		t.Errorf("Expected balance 10000, got %d", acc.Balance)
	}

	list := svc.DebugGetPatternList()
	if len(list) == 0 {
		t.Error("Expected pattern list, got empty")
	}
}

func TestTutorialProgress(t *testing.T) {
	_, eco := setupTestDB(t)
	defer teardownTestDB(t, eco)

	svc := NewTradingService(eco, "pwd", 500, 100)
	jid := "628123456789@s.whatsapp.net"

	progress, err := svc.GetTutorialProgress(jid)
	if err != nil {
		t.Fatalf("GetTutorialProgress error: %v", err)
	}
	if progress.CompletedSteps != 0 {
		t.Errorf("Expected completed steps 0, got %d", progress.CompletedSteps)
	}
	if progress.PracticeDone {
		t.Error("Expected practice done false, got true")
	}

	err = svc.CompleteTutorialStep(jid, 3)
	if err != nil {
		t.Fatalf("CompleteTutorialStep error: %v", err)
	}

	progress, _ = svc.GetTutorialProgress(jid)
	if progress.CompletedSteps != 3 {
		t.Errorf("Expected completed steps 3, got %d", progress.CompletedSteps)
	}

	err = svc.CompletePractice(jid)
	if err != nil {
		t.Fatalf("CompletePractice error: %v", err)
	}

	progress, _ = svc.GetTutorialProgress(jid)
	if !progress.PracticeDone {
		t.Error("Expected practice done true, got false")
	}
	if progress.CompletedSteps != len(TutorialSteps) {
		t.Errorf("Expected all steps completed (%d), got %d", len(TutorialSteps), progress.CompletedSteps)
	}
}

func TestSessionTimeouts(t *testing.T) {
	_, eco := setupTestDB(t)
	defer teardownTestDB(t, eco)

	svc := NewTradingService(eco, "pwd", 500, 100)
	jid := "628123456789@s.whatsapp.net"

	_, sessionID, err := svc.StartSession(jid)
	if err != nil {
		t.Fatalf("StartSession error: %v", err)
	}

	// Verify active session in memory
	sess, ok := svc.GetActiveSession(jid)
	if !ok || sess.ID != sessionID {
		t.Error("Active session not cached properly")
	}

	// Simulate auto close by calling internal close
	svc.endSessionInternal(jid, sess)

	// Verify closed
	_, ok = svc.GetActiveSession(jid)
	if ok {
		t.Error("Session was not closed and deleted from active cache")
	}
}

func TestOpenPositionRejectsInvalidEntryPrice(t *testing.T) {
	_, eco := setupTestDB(t)
	defer teardownTestDB(t, eco)

	svc := NewTradingService(eco, "pwd", 500, 100)
	jid := "628123456789@s.whatsapp.net"

	if err := eco.AddBalance(jid, 5000, "test", "setup"); err != nil {
		t.Fatalf("AddBalance error: %v", err)
	}
	if err := svc.Deposit(jid, 1000); err != nil {
		t.Fatalf("Deposit error: %v", err)
	}

	_, sessionID, err := svc.StartSession(jid)
	if err != nil {
		t.Fatalf("StartSession error: %v", err)
	}

	invalidPrices := []float64{0, -1, math.Inf(1), math.NaN()}
	for _, price := range invalidPrices {
		req := OpenPositionRequest{
			SessionID:  sessionID,
			Direction:  DirBearish,
			Leverage:   3,
			Size:       100,
			EntryPrice: price,
		}
		if _, err := svc.OpenPosition(jid, req); !errors.Is(err, ErrInvalidPrice) {
			t.Fatalf("Expected ErrInvalidPrice for entry price %v, got %v", price, err)
		}
	}

	acc, _ := svc.GetTradingBalance(jid)
	if acc.Balance != 1000 {
		t.Errorf("Invalid open attempts should not change balance, got %d", acc.Balance)
	}
}

func TestTradingRiskManagementRules(t *testing.T) {
	_, eco := setupTestDB(t)
	defer teardownTestDB(t, eco)

	svc := NewTradingService(eco, "pwd", 500, 100)
	jid := "628123456789@s.whatsapp.net"

	if err := eco.AddBalance(jid, 5000, "test", "setup"); err != nil {
		t.Fatalf("AddBalance error: %v", err)
	}
	if err := svc.Deposit(jid, 1000); err != nil {
		t.Fatalf("Deposit error: %v", err)
	}

	_, sessionID, err := svc.StartSession(jid)
	if err != nil {
		t.Fatalf("StartSession error: %v", err)
	}

	marketPrice := 100.0
	req := OpenPositionRequest{
		SessionID:  sessionID,
		Direction:  DirBullish,
		Leverage:   2,
		Size:       100,
		EntryPrice: marketPrice,
	}

	pos, err := svc.OpenPosition(jid, req)
	if err != nil {
		t.Fatalf("OpenPosition error: %v", err)
	}
	if pos.EntryPrice <= marketPrice {
		t.Fatalf("Expected long entry spread above market %.2f, got %.2f", marketPrice, pos.EntryPrice)
	}
	if pos.Fee != calculateTradingFee(req.Size, req.Leverage) {
		t.Fatalf("Expected fee %d, got %d", calculateTradingFee(req.Size, req.Leverage), pos.Fee)
	}

	if _, err := svc.OpenPosition(jid, req); !errors.Is(err, ErrTradeCooldown) {
		t.Fatalf("Expected cooldown error after immediate re-open, got %v", err)
	}

	closed, err := svc.ClosePosition(jid, pos.ID, pos.EntryPrice)
	if err != nil {
		t.Fatalf("ClosePosition error: %v", err)
	}
	if closed.PnL != -pos.Fee {
		t.Fatalf("Expected flat close to lose fee %d, got %d", pos.Fee, closed.PnL)
	}
}

func TestLeaderboard(t *testing.T) {
	_, eco := setupTestDB(t)
	defer teardownTestDB(t, eco)

	svc := NewTradingService(eco, "pwd", 500, 100)

	// Create two users with different trading balances
	jid1 := "user1@s.whatsapp.net"
	jid2 := "user2@s.whatsapp.net"

	// Initialize balances
	_, _ = svc.db.Exec("INSERT OR REPLACE INTO trading_balances (jid, balance, total_profit, total_loss, total_trades, total_sessions) VALUES (?, 1500, 2000, 500, 10, 2)", jid1)
	_, _ = svc.db.Exec("INSERT OR REPLACE INTO trading_balances (jid, balance, total_profit, total_loss, total_trades, total_sessions) VALUES (?, 3000, 3500, 500, 15, 3)", jid2)

	// Query leaderboard
	list, err := svc.GetLeaderboard(10)
	if err != nil {
		t.Fatalf("GetLeaderboard error: %v", err)
	}

	if len(list) != 2 {
		t.Fatalf("Expected 2 leaderboard entries, got %d", len(list))
	}

	// Should be sorted by balance descending: user2 (3000) then user1 (1500)
	if list[0].Username != "user2" || list[0].Balance != 3000 {
		t.Errorf("Expected rank 1 to be user2 with balance 3000, got %s with %d", list[0].Username, list[0].Balance)
	}
	if list[1].Username != "user1" || list[1].Balance != 1500 {
		t.Errorf("Expected rank 2 to be user1 with balance 1500, got %s with %d", list[1].Username, list[1].Balance)
	}
}
