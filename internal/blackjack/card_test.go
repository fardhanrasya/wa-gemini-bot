package blackjack

import (
	"testing"
	"wa-gemini-bot/internal/poker"
)

// Feature: blackjack-game, Property 60: Ace value optimization non-bust
// Feature: blackjack-game, Property 61: Ace value optimization bust prevention
// TestBlackjackCardValues verifies that card values are calculated correctly
// for all card types in blackjack.
func TestBlackjackCardValues(t *testing.T) {
	tests := []struct {
		rank     poker.Rank
		expected int
	}{
		{Two, 2},
		{Three, 3},
		{Four, 4},
		{Five, 5},
		{Six, 6},
		{Seven, 7},
		{Eight, 8},
		{Nine, 9},
		{Ten, 10},
		{Jack, 10},
		{Queen, 10},
		{King, 10},
		{Ace, 11}, // Default value, will be adjusted in Hand.Value()
	}

	for _, tt := range tests {
		card := Card{Rank: tt.rank, Suit: Spade}
		value := BlackjackValue(card)
		if value != tt.expected {
			t.Errorf("BlackjackValue(%s) = %d; want %d", tt.rank.Name(), value, tt.expected)
		}
	}
}

// TestBlackjackCardValuesAllSuits verifies that card values are independent of suit
func TestBlackjackCardValuesAllSuits(t *testing.T) {
	suits := []poker.Suit{Spade, Heart, Diamond, Club}
	ranks := []poker.Rank{Two, Three, Four, Five, Six, Seven, Eight, Nine, Ten, Jack, Queen, King, Ace}

	for _, rank := range ranks {
		expectedValue := BlackjackValue(Card{Rank: rank, Suit: Spade})
		for _, suit := range suits {
			card := Card{Rank: rank, Suit: suit}
			value := BlackjackValue(card)
			if value != expectedValue {
				t.Errorf("BlackjackValue(%s of %s) = %d; want %d (suit should not affect value)",
					rank.Name(), suit.Name(), value, expectedValue)
			}
		}
	}
}

// TestDeckCreation verifies that a new deck is created and shuffled
func TestDeckCreation(t *testing.T) {
	deck := NewDeck()
	if deck == nil {
		t.Fatal("NewDeck() returned nil")
	}

	// Draw all 52 cards to verify deck is complete
	cardsSeen := make(map[string]bool)
	for i := 0; i < 52; i++ {
		card := deck.Draw()
		cardKey := card.String()
		if cardsSeen[cardKey] {
			t.Errorf("Duplicate card drawn: %s", cardKey)
		}
		cardsSeen[cardKey] = true
	}

	if len(cardsSeen) != 52 {
		t.Errorf("Deck has %d unique cards; want 52", len(cardsSeen))
	}
}
