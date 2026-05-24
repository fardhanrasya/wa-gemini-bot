package economy

import (
	"os"
	"testing"
)

func TestGetRankByBalance(t *testing.T) {
	tests := []struct {
		balance  int
		expected string
	}{
		{0, "Peasant"},
		{4999, "Peasant"},
		{5000, "Levy"},
		{19999, "Levy"},
		{20000, "Mercenary"},
		{50000, "Governor"},
		{100000, "Diplomat"},
		{250000, "General"},
		{500000, "Strategist"},
		{1000000, "Monarch"},
		{5000000, "Emperor"},
		{10000000, "Hegemon"},
		{50000000, "Hegemon"},
	}

	for _, tt := range tests {
		rank := GetRankByBalance(tt.balance)
		if rank.Name != tt.expected {
			t.Errorf("GetRankByBalance(%d) = %s; want %s", tt.balance, rank.Name, tt.expected)
		}
	}
}

func TestCustomNameSystem(t *testing.T) {
	dbPath := "test-economy.db"
	defer os.Remove(dbPath)

	s, err := NewEconomyService(dbPath)
	if err != nil {
		t.Fatalf("gagal membuat service: %v", err)
	}
	defer s.Close()

	jid := "628123456789@s.whatsapp.net"

	// 1. Awalnya belum ada user, register via GetBalance
	_, err = s.GetBalance(jid)
	if err != nil {
		t.Fatalf("gagal get balance: %v", err)
	}

	// 2. Set WA name
	err = s.UpdateName(jid, "WhatsApp Name")
	if err != nil {
		t.Fatalf("gagal update name: %v", err)
	}

	user, err := s.GetUser(jid)
	if err != nil {
		t.Fatalf("gagal get user: %v", err)
	}
	if user.Name != "WhatsApp Name" {
		t.Errorf("user name = %s; want WhatsApp Name", user.Name)
	}

	// 3. Set custom name
	err = s.SetCustomName(jid, "Custom Nickname")
	if err != nil {
		t.Fatalf("gagal set custom name: %v", err)
	}

	user, err = s.GetUser(jid)
	if err != nil {
		t.Fatalf("gagal get user: %v", err)
	}
	if user.Name != "Custom Nickname" {
		t.Errorf("user name = %s; want Custom Nickname", user.Name)
	}

	// 4. UpdateName shouldn't overwrite custom name
	err = s.UpdateName(jid, "Another WA Name")
	if err != nil {
		t.Fatalf("gagal update name: %v", err)
	}

	user, err = s.GetUser(jid)
	if err != nil {
		t.Fatalf("gagal get user: %v", err)
	}
	if user.Name != "Custom Nickname" {
		t.Errorf("user name = %s; want Custom Nickname (protected)", user.Name)
	}

	// 5. Reset custom name
	err = s.ResetCustomName(jid)
	if err != nil {
		t.Fatalf("gagal reset custom name: %v", err)
	}

	// 6. UpdateName should now succeed in updating the name
	err = s.UpdateName(jid, "New WA Name")
	if err != nil {
		t.Fatalf("gagal update name: %v", err)
	}

	user, err = s.GetUser(jid)
	if err != nil {
		t.Fatalf("gagal get user: %v", err)
	}
	if user.Name != "New WA Name" {
		t.Errorf("user name = %s; want New WA Name (unlocked)", user.Name)
	}
}

// Feature: blackjack-game, Property 1: Dealer account exists after initialization
// TestDealerAccountInitialization verifies that the dealer account is created
// with correct JID, name, and initial balance when economy service is initialized.
func TestDealerAccountInitialization(t *testing.T) {
	dbPath := "test-dealer-init.db"
	defer os.Remove(dbPath)

	// Create new economy service (should auto-initialize dealer)
	s, err := NewEconomyService(dbPath)
	if err != nil {
		t.Fatalf("gagal membuat service: %v", err)
	}
	defer s.Close()

	// Verify dealer account exists
	balance, err := s.GetBalance(DealerJID)
	if err != nil {
		t.Fatalf("dealer account tidak ditemukan: %v", err)
	}

	// Verify dealer has correct initial balance
	if balance != DealerInitialBalance {
		t.Errorf("dealer balance = %d; want %d", balance, DealerInitialBalance)
	}

	// Verify dealer user info
	user, err := s.GetUser(DealerJID)
	if err != nil {
		t.Fatalf("gagal get dealer user: %v", err)
	}

	if user.JID != DealerJID {
		t.Errorf("dealer JID = %s; want %s", user.JID, DealerJID)
	}

	if user.Name != DealerName {
		t.Errorf("dealer name = %s; want %s", user.Name, DealerName)
	}

	if user.Balance != DealerInitialBalance {
		t.Errorf("dealer balance = %d; want %d", user.Balance, DealerInitialBalance)
	}

	// Verify dealer account is idempotent (calling init again should not error)
	err = s.InitializeDealerAccount()
	if err != nil {
		t.Errorf("re-initializing dealer account should not error: %v", err)
	}

	// Balance should remain the same
	balanceAfter, err := s.GetBalance(DealerJID)
	if err != nil {
		t.Fatalf("gagal get balance after re-init: %v", err)
	}

	if balanceAfter != DealerInitialBalance {
		t.Errorf("dealer balance after re-init = %d; want %d", balanceAfter, DealerInitialBalance)
	}
}

// TestGetLeaderboard_ExcludesDealer verifies that the dealer account is not included in the leaderboard.
func TestGetLeaderboard_ExcludesDealer(t *testing.T) {
	dbPath := "test-leaderboard-dealer.db"
	defer os.Remove(dbPath)

	s, err := NewEconomyService(dbPath)
	if err != nil {
		t.Fatalf("gagal membuat service: %v", err)
	}
	defer s.Close()

	// Register some normal players
	player1 := "player1@test.com"
	player2 := "player2@test.com"

	// normal registration gives InitialBalance (5000), let's add some balance
	s.AddBalance(player1, 10000, "test", "ref")
	s.AddBalance(player2, 20000, "test", "ref")

	// Get leaderboard
	leaderboard, err := s.GetLeaderboard(10)
	if err != nil {
		t.Fatalf("gagal mengambil leaderboard: %v", err)
	}

	// Verify dealer is not in the leaderboard
	for _, user := range leaderboard {
		if user.JID == DealerJID {
			t.Errorf("dealer JID (%s) should be excluded from the leaderboard", DealerJID)
		}
	}
}

