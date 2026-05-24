package blackjack

// ==========================================================================
// Hand — represents a blackjack hand with flexible Ace handling.
//
// Key features:
// - Automatic Ace value optimization (11 or 1)
// - Blackjack detection (Ace + 10-value card in 2 cards)
// - Bust detection (value > 21)
// - Soft hand detection (Ace counted as 11)
// ==========================================================================

// Hand represents a collection of cards in blackjack
type Hand struct {
	Cards []Card
}

// NewHand creates a new empty hand
func NewHand() *Hand {
	return &Hand{
		Cards: make([]Card, 0),
	}
}

// AddCard adds a card to the hand
func (h *Hand) AddCard(card Card) {
	h.Cards = append(h.Cards, card)
}

// Value calculates the optimal value of the hand.
// Aces are counted as 11 if possible, otherwise as 1.
// At most one Ace is counted as 11 to avoid busting.
func (h *Hand) Value() int {
	total := 0
	aces := 0

	// First pass: count all cards, Aces as 11
	for _, card := range h.Cards {
		value := BlackjackValue(card)
		if card.Rank == Ace {
			aces++
		}
		total += value
	}

	// Second pass: convert Aces from 11 to 1 if needed to avoid bust
	// Each conversion reduces total by 10 (11 - 1 = 10)
	for aces > 0 && total > 21 {
		total -= 10
		aces--
	}

	return total
}

// IsBlackjack returns true if the hand is a natural blackjack
// (Ace + 10-value card in exactly 2 cards)
func (h *Hand) IsBlackjack() bool {
	if len(h.Cards) != 2 {
		return false
	}

	// Check if one card is Ace and the other is 10-value
	hasAce := false
	hasTen := false

	for _, card := range h.Cards {
		if card.Rank == Ace {
			hasAce = true
		}
		if BlackjackValue(card) == 10 {
			hasTen = true
		}
	}

	return hasAce && hasTen
}

// IsBust returns true if the hand value exceeds 21
func (h *Hand) IsBust() bool {
	return h.Value() > 21
}

// IsSoft returns true if the hand contains an Ace counted as 11
// (i.e., the hand value would decrease by 10 if the Ace was counted as 1)
func (h *Hand) IsSoft() bool {
	if len(h.Cards) == 0 {
		return false
	}

	// Check if hand has an Ace
	hasAce := false
	for _, card := range h.Cards {
		if card.Rank == Ace {
			hasAce = true
			break
		}
	}

	if !hasAce {
		return false
	}

	// Calculate value with all Aces as 1
	hardValue := 0
	for _, card := range h.Cards {
		if card.Rank == Ace {
			hardValue += 1
		} else {
			hardValue += BlackjackValue(card)
		}
	}

	// If current value is 11 more than hard value, we have a soft Ace
	return h.Value() == hardValue+10
}

// Clear removes all cards from the hand
func (h *Hand) Clear() {
	h.Cards = h.Cards[:0]
}

// CardCount returns the number of cards in the hand
func (h *Hand) CardCount() int {
	return len(h.Cards)
}

// String returns a string representation of the hand
func (h *Hand) String() string {
	if len(h.Cards) == 0 {
		return "Empty hand"
	}
	return RenderCardsShort(h.Cards)
}
