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
