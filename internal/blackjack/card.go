package blackjack

import (
	"wa-gemini-bot/internal/poker"
)

// ==========================================================================
// Blackjack card system — reuses poker card structure with blackjack-specific
// value calculation.
//
// Card values in blackjack:
// - Number cards (2-10): face value
// - Face cards (J, Q, K): 10
// - Ace: 11 (default), adjusted to 1 in Hand.Value() if needed
// ==========================================================================

// Card is an alias for poker.Card to reuse the existing card structure
type Card = poker.Card

// Deck is an alias for poker.Deck to reuse deck management
type Deck = poker.Deck

// Rank constants from poker
const (
	Two   = poker.Two
	Three = poker.Three
	Four  = poker.Four
	Five  = poker.Five
	Six   = poker.Six
	Seven = poker.Seven
	Eight = poker.Eight
	Nine  = poker.Nine
	Ten   = poker.Ten
	Jack  = poker.Jack
	Queen = poker.Queen
	King  = poker.King
	Ace   = poker.Ace
)

// Suit constants from poker
const (
	Spade   = poker.Spade
	Heart   = poker.Heart
	Diamond = poker.Diamond
	Club    = poker.Club
)

// NewDeck creates a new shuffled deck for blackjack
func NewDeck() *Deck {
	return poker.NewDeck()
}

// BlackjackValue returns the blackjack value of a card.
// Ace returns 11 by default (will be adjusted to 1 in Hand.Value() if needed).
// Face cards (J, Q, K) return 10.
// Number cards return their face value.
func BlackjackValue(c Card) int {
	switch c.Rank {
	case Ace:
		return 11 // Default to 11, adjusted in Hand.Value()
	case Jack, Queen, King:
		return 10
	default:
		return int(c.Rank)
	}
}

// RenderCards renders cards for WhatsApp display
func RenderCards(cards []Card) string {
	return poker.RenderCards(cards)
}

// RenderCardsShort renders cards in short format
func RenderCardsShort(cards []Card) string {
	return poker.RenderCardsShort(cards)
}
