package blackjack

import "testing"

// TestStickyBetPreservedAcrossRounds verifies sticky bet behavior and all-in when short on chips.
func TestStickyBetPreservedAcrossRounds(t *testing.T) {
	game := NewBlackjackGame()
	_ = game.AddPlayer("Alice", "alice@test.com", 6000)

	// Ronde 1: bet 6000, deduct -> chips 0 during play
	deck1 := NewDeck()
	deck1.SetCards([]Card{
		{Rank: Ten, Suit: Spade}, {Rank: Ten, Suit: Diamond},
		{Rank: Five, Suit: Spade}, {Rank: Seven, Suit: Diamond},
	})
	game.Deck = deck1
	_, err := game.StartRound()
	if err != nil {
		t.Fatalf("StartRound: %v", err)
	}
	alice := game.GetPlayer("Alice")
	if alice.Bet != 6000 || alice.Chips != 0 {
		t.Fatalf("after round 1 start: bet=%d chips=%d; want bet=6000 chips=0", alice.Bet, alice.Chips)
	}

	// Simulasikan menang: payout 12000 (2x bet)
	alice.Chips += 12000
	game.Reset()

	if alice.Bet != 6000 {
		t.Errorf("after Reset sticky bet = %d; want 6000", alice.Bet)
	}
	if alice.Chips != 12000 {
		t.Errorf("after win chips = %d; want 12000", alice.Chips)
	}

	// Ronde 2: sticky bet 6000, sisa 6000 di meja
	deck2 := NewDeck()
	deck2.SetCards([]Card{
		{Rank: Ten, Suit: Spade}, {Rank: Ten, Suit: Diamond},
		{Rank: Five, Suit: Spade}, {Rank: Seven, Suit: Diamond},
	})
	game.Deck = deck2
	info, err := game.StartRound()
	if err != nil {
		t.Fatalf("StartRound 2: %v", err)
	}
	if len(info.BetAdjustments) != 0 {
		t.Errorf("unexpected bet adjustments: %+v", info.BetAdjustments)
	}
	if alice.Bet != 6000 || alice.Chips != 6000 {
		t.Fatalf("after round 2 start: bet=%d chips=%d; want bet=6000 chips=6000", alice.Bet, alice.Chips)
	}

	// Edge case: saldo meja 2000, sticky bet 6000 -> all-in 2000
	alice.Chips = 2000
	alice.Bet = 6000
	game.Reset()
	deck3 := NewDeck()
	deck3.SetCards([]Card{
		{Rank: Ten, Suit: Spade}, {Rank: Ten, Suit: Diamond},
		{Rank: Five, Suit: Spade}, {Rank: Seven, Suit: Diamond},
	})
	game.Deck = deck3
	info, err = game.StartRound()
	if err != nil {
		t.Fatalf("StartRound 3: %v", err)
	}
	if len(info.BetAdjustments) != 1 {
		t.Fatalf("bet adjustments = %d; want 1", len(info.BetAdjustments))
	}
	adj := info.BetAdjustments[0]
	if adj.OldBet != 6000 || adj.NewBet != 2000 {
		t.Errorf("adjustment = %+v; want 6000 -> 2000", adj)
	}
	if alice.Bet != 2000 || alice.Chips != 0 {
		t.Fatalf("after all-in start: bet=%d chips=%d; want bet=2000 chips=0", alice.Bet, alice.Chips)
	}
}
