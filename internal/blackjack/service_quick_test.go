package blackjack

import "testing"

func TestQuickCommandRegex(t *testing.T) {
	tests := []struct {
		text    string
		isBet   bool
		isLeave bool
	}{
		{"bet 500", true, false},
		{"BET 10000", true, false},
		{"taruhan 200", true, false},
		{"leave", false, true},
		{"keluar", false, true},
		{"hit", false, false},
		{"bet", false, false},
		{"bet abc", false, false},
	}

	for _, tt := range tests {
		betMatch := quickBetRegex.FindStringSubmatch(tt.text)
		isBet := betMatch != nil
		isLeave := quickLeaveRegex.MatchString(tt.text)
		if isBet != tt.isBet {
			t.Errorf("quickBetRegex(%q) = %v; want %v", tt.text, isBet, tt.isBet)
		}
		if isLeave != tt.isLeave {
			t.Errorf("quickLeaveRegex(%q) = %v; want %v", tt.text, isLeave, tt.isLeave)
		}
	}
}

func TestHandleQuickCommandRequiresPlayerInGame(t *testing.T) {
	s := NewBlackjackService(60, 10)
	groupJID := "test@g.us"

	if s.HandleQuickCommand(groupJID, "Alice", "alice@test.com", "bet 100") {
		t.Error("should not handle bet when no session")
	}
	if s.HandleQuickCommand(groupJID, "Alice", "alice@test.com", "leave") {
		t.Error("should not handle leave when no session")
	}
}
