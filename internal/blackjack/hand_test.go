package blackjack

import (
	"testing"
)

// Feature: blackjack-game, Property 62: Multiple Ace handling
// TestHandValueWithMultipleAces verifies that at most one Ace is counted as 11
func TestHandValueWithMultipleAces(t *testing.T) {
	tests := []struct {
		name     string
		cards    []Card
		expected int
	}{
		{
			name:     "Single Ace",
			cards:    []Card{{Rank: Ace, Suit: Spade}},
			expected: 11,
		},
		{
			name:     "Ace + 5",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Five, Suit: Heart}},
			expected: 16, // 11 + 5
		},
		{
			name:     "Ace + 10 (Blackjack)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: King, Suit: Heart}},
			expected: 21, // 11 + 10
		},
		{
			name:     "Ace + Ace",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Ace, Suit: Heart}},
			expected: 12, // 11 + 1 (only one Ace as 11)
		},
		{
			name:     "Ace + Ace + 9",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Ace, Suit: Heart}, {Rank: Nine, Suit: Diamond}},
			expected: 21, // 11 + 1 + 9
		},
		{
			name:     "Three Aces",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Ace, Suit: Heart}, {Rank: Ace, Suit: Diamond}},
			expected: 13, // 11 + 1 + 1
		},
		{
			name:     "Four Aces",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Ace, Suit: Heart}, {Rank: Ace, Suit: Diamond}, {Rank: Ace, Suit: Club}},
			expected: 14, // 11 + 1 + 1 + 1
		},
		{
			name:     "Ace + 6 + 4 (Ace as 1)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Six, Suit: Heart}, {Rank: Four, Suit: Diamond}},
			expected: 21, // 11 + 6 + 4 = 21
		},
		{
			name:     "Ace + 6 + 5 (Ace must be 1)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Six, Suit: Heart}, {Rank: Five, Suit: Diamond}},
			expected: 12, // 1 + 6 + 5 (Ace as 1 to avoid bust)
		},
		{
			name:     "Ace + King + 5 (Bust with Ace as 11, so Ace becomes 1)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: King, Suit: Heart}, {Rank: Five, Suit: Diamond}},
			expected: 16, // 1 + 10 + 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := NewHand()
			for _, card := range tt.cards {
				hand.AddCard(card)
			}
			value := hand.Value()
			if value != tt.expected {
				t.Errorf("Hand value = %d; want %d (cards: %v)", value, tt.expected, tt.cards)
			}
		})
	}
}

// TestHandValueNoAces verifies hand values without Aces
func TestHandValueNoAces(t *testing.T) {
	tests := []struct {
		name     string
		cards    []Card
		expected int
	}{
		{
			name:     "Single card",
			cards:    []Card{{Rank: Five, Suit: Spade}},
			expected: 5,
		},
		{
			name:     "Two cards",
			cards:    []Card{{Rank: Seven, Suit: Spade}, {Rank: Eight, Suit: Heart}},
			expected: 15,
		},
		{
			name:     "Face cards",
			cards:    []Card{{Rank: King, Suit: Spade}, {Rank: Queen, Suit: Heart}, {Rank: Jack, Suit: Diamond}},
			expected: 30,
		},
		{
			name:     "Mixed",
			cards:    []Card{{Rank: Two, Suit: Spade}, {Rank: Three, Suit: Heart}, {Rank: Four, Suit: Diamond}},
			expected: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := NewHand()
			for _, card := range tt.cards {
				hand.AddCard(card)
			}
			value := hand.Value()
			if value != tt.expected {
				t.Errorf("Hand value = %d; want %d", value, tt.expected)
			}
		})
	}
}

// Feature: blackjack-game, Property 64: Blackjack detection
// TestIsBlackjack verifies blackjack detection
func TestIsBlackjack(t *testing.T) {
	tests := []struct {
		name     string
		cards    []Card
		expected bool
	}{
		{
			name:     "Ace + King",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: King, Suit: Heart}},
			expected: true,
		},
		{
			name:     "Ace + Queen",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Queen, Suit: Heart}},
			expected: true,
		},
		{
			name:     "Ace + Jack",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Jack, Suit: Heart}},
			expected: true,
		},
		{
			name:     "Ace + 10",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Ten, Suit: Heart}},
			expected: true,
		},
		{
			name:     "King + Ace (order doesn't matter)",
			cards:    []Card{{Rank: King, Suit: Spade}, {Rank: Ace, Suit: Heart}},
			expected: true,
		},
		{
			name:     "Ace + 9 (not blackjack)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Nine, Suit: Heart}},
			expected: false,
		},
		{
			name:     "King + Queen (not blackjack)",
			cards:    []Card{{Rank: King, Suit: Spade}, {Rank: Queen, Suit: Heart}},
			expected: false,
		},
		{
			name:     "Three cards totaling 21 (not blackjack)",
			cards:    []Card{{Rank: Seven, Suit: Spade}, {Rank: Seven, Suit: Heart}, {Rank: Seven, Suit: Diamond}},
			expected: false,
		},
		{
			name:     "Ace + 5 + 5 (21 but not blackjack)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Five, Suit: Heart}, {Rank: Five, Suit: Diamond}},
			expected: false,
		},
		{
			name:     "Single card",
			cards:    []Card{{Rank: Ace, Suit: Spade}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := NewHand()
			for _, card := range tt.cards {
				hand.AddCard(card)
			}
			isBlackjack := hand.IsBlackjack()
			if isBlackjack != tt.expected {
				t.Errorf("IsBlackjack() = %v; want %v (cards: %v)", isBlackjack, tt.expected, tt.cards)
			}
		})
	}
}

// TestIsBust verifies bust detection
func TestIsBust(t *testing.T) {
	tests := []struct {
		name     string
		cards    []Card
		expected bool
	}{
		{
			name:     "Bust with 22",
			cards:    []Card{{Rank: King, Suit: Spade}, {Rank: Queen, Suit: Heart}, {Rank: Two, Suit: Diamond}},
			expected: true,
		},
		{
			name:     "Bust with 25",
			cards:    []Card{{Rank: King, Suit: Spade}, {Rank: Queen, Suit: Heart}, {Rank: Five, Suit: Diamond}},
			expected: true,
		},
		{
			name:     "Not bust with 21",
			cards:    []Card{{Rank: King, Suit: Spade}, {Rank: Queen, Suit: Heart}, {Rank: Ace, Suit: Diamond}},
			expected: false,
		},
		{
			name:     "Not bust with 20",
			cards:    []Card{{Rank: King, Suit: Spade}, {Rank: Queen, Suit: Heart}},
			expected: false,
		},
		{
			name:     "Not bust with Ace adjustment",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: King, Suit: Heart}, {Rank: Five, Suit: Diamond}},
			expected: false, // 1 + 10 + 5 = 16
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := NewHand()
			for _, card := range tt.cards {
				hand.AddCard(card)
			}
			isBust := hand.IsBust()
			if isBust != tt.expected {
				t.Errorf("IsBust() = %v; want %v (value: %d)", isBust, tt.expected, hand.Value())
			}
		})
	}
}

// TestIsSoft verifies soft hand detection
func TestIsSoft(t *testing.T) {
	tests := []struct {
		name     string
		cards    []Card
		expected bool
	}{
		{
			name:     "Soft 17 (Ace + 6)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Six, Suit: Heart}},
			expected: true,
		},
		{
			name:     "Soft 18 (Ace + 7)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Seven, Suit: Heart}},
			expected: true,
		},
		{
			name:     "Soft 21 (Ace + King)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: King, Suit: Heart}},
			expected: true,
		},
		{
			name:     "Hard 17 (10 + 7)",
			cards:    []Card{{Rank: Ten, Suit: Spade}, {Rank: Seven, Suit: Heart}},
			expected: false,
		},
		{
			name:     "Hard 12 with Ace (Ace + 6 + 5)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Six, Suit: Heart}, {Rank: Five, Suit: Diamond}},
			expected: false, // Ace must be 1, so it's hard
		},
		{
			name:     "Soft 21 with three cards (Ace + 5 + 5)",
			cards:    []Card{{Rank: Ace, Suit: Spade}, {Rank: Five, Suit: Heart}, {Rank: Five, Suit: Diamond}},
			expected: true,
		},
		{
			name:     "No Ace",
			cards:    []Card{{Rank: King, Suit: Spade}, {Rank: Queen, Suit: Heart}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hand := NewHand()
			for _, card := range tt.cards {
				hand.AddCard(card)
			}
			isSoft := hand.IsSoft()
			if isSoft != tt.expected {
				t.Errorf("IsSoft() = %v; want %v (value: %d)", isSoft, tt.expected, hand.Value())
			}
		})
	}
}

// TestHandOperations verifies basic hand operations
func TestHandOperations(t *testing.T) {
	hand := NewHand()

	// Test empty hand
	if hand.CardCount() != 0 {
		t.Errorf("New hand should have 0 cards, got %d", hand.CardCount())
	}

	// Add cards
	hand.AddCard(Card{Rank: Ace, Suit: Spade})
	hand.AddCard(Card{Rank: King, Suit: Heart})

	if hand.CardCount() != 2 {
		t.Errorf("Hand should have 2 cards, got %d", hand.CardCount())
	}

	// Clear hand
	hand.Clear()
	if hand.CardCount() != 0 {
		t.Errorf("Cleared hand should have 0 cards, got %d", hand.CardCount())
	}
}
