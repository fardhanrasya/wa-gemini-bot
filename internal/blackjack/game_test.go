package blackjack

import (
	"fmt"
	"testing"
)

// Feature: blackjack-game, Property 12: Maximum player limit
// TestMaxPlayerLimit verifies that the 8th player is rejected
func TestMaxPlayerLimit(t *testing.T) {
	game := NewBlackjackGame()

	// Add 7 players (max)
	for i := 1; i <= MaxPlayers; i++ {
		err := game.AddPlayer(
			fmt.Sprintf("Player%d", i),
			fmt.Sprintf("player%d@test.com", i),
			100,
		)
		if err != nil {
			t.Fatalf("Failed to add player %d: %v", i, err)
		}
	}

	// Verify we have 7 players
	if game.PlayerCount() != MaxPlayers {
		t.Errorf("Player count = %d; want %d", game.PlayerCount(), MaxPlayers)
	}

	// Try to add 8th player (should fail)
	err := game.AddPlayer("Player8", "player8@test.com", 100)
	if err != ErrMaxPlayersReached {
		t.Errorf("Adding 8th player should return ErrMaxPlayersReached, got: %v", err)
	}

	// Verify still 7 players
	if game.PlayerCount() != MaxPlayers {
		t.Errorf("Player count after rejected add = %d; want %d", game.PlayerCount(), MaxPlayers)
	}
}

// Feature: blackjack-game, Property 9: Duplicate join rejection
// TestDuplicatePlayerRejection verifies that duplicate players are rejected
func TestDuplicatePlayerRejection(t *testing.T) {
	game := NewBlackjackGame()

	// Add first player
	err := game.AddPlayer("Alice", "alice@test.com", 100)
	if err != nil {
		t.Fatalf("Failed to add first player: %v", err)
	}

	// Try to add same player by name
	err = game.AddPlayer("Alice", "different@test.com", 100)
	if err != ErrPlayerExists {
		t.Errorf("Adding duplicate name should return ErrPlayerExists, got: %v", err)
	}

	// Try to add same player by JID
	err = game.AddPlayer("Different", "alice@test.com", 100)
	if err != ErrPlayerExists {
		t.Errorf("Adding duplicate JID should return ErrPlayerExists, got: %v", err)
	}

	// Verify still only 1 player
	if game.PlayerCount() != 1 {
		t.Errorf("Player count = %d; want 1", game.PlayerCount())
	}
}

// TestAddPlayerOnlyInLobby verifies players can join in lobby/finished but not mid-round.
func TestAddPlayerOnlyInLobby(t *testing.T) {
	game := NewBlackjackGame()

	// Should work in lobby
	err := game.AddPlayer("Alice", "alice@test.com", 100)
	if err != nil {
		t.Fatalf("Failed to add player in lobby: %v", err)
	}

	// Should work in finished phase (buy-in ulang / jeda antar ronde)
	game.Phase = PhaseFinished
	err = game.AddPlayer("Bob", "bob@test.com", 100)
	if err != nil {
		t.Errorf("Adding player in finished phase should succeed, got: %v", err)
	}

	// Should fail mid-round
	game.Phase = PhasePlayerTurns
	err = game.AddPlayer("Charlie", "charlie@test.com", 100)
	if err != ErrGameNotInLobby {
		t.Errorf("Adding player mid-round should return ErrGameNotInLobby, got: %v", err)
	}
}

// TestRebuyAfterBust simulates kicked player re-joining empty table in grace/finished phase.
func TestRebuyAfterBust(t *testing.T) {
	game := NewBlackjackGame()
	_ = game.AddPlayer("Alice", "alice@test.com", 100)
	game.Phase = PhaseFinished
	game.RemovePlayer("Alice")

	if game.PlayerCount() != 0 {
		t.Fatalf("player count = %d; want 0", game.PlayerCount())
	}

	err := game.AddPlayer("Alice", "alice@test.com", 200)
	if err != nil {
		t.Fatalf("re-buy after bust should succeed, got: %v", err)
	}
	alice := game.GetPlayer("Alice")
	if alice == nil || alice.Chips != 200 || alice.Bet != 200 {
		t.Fatalf("rebuy player state wrong: %+v", alice)
	}
}

// TestGetPlayer verifies player retrieval
func TestGetPlayer(t *testing.T) {
	game := NewBlackjackGame()

	// Add players
	game.AddPlayer("Alice", "alice@test.com", 100)
	game.AddPlayer("Bob", "bob@test.com", 200)

	// Get by name
	alice := game.GetPlayer("Alice")
	if alice == nil {
		t.Fatal("GetPlayer(Alice) returned nil")
	}
	if alice.Name != "Alice" || alice.JID != "alice@test.com" || alice.Bet != 100 {
		t.Errorf("Alice data incorrect: %+v", alice)
	}

	// Get by JID
	bob := game.GetPlayerByJID("bob@test.com")
	if bob == nil {
		t.Fatal("GetPlayerByJID(bob@test.com) returned nil")
	}
	if bob.Name != "Bob" || bob.Bet != 200 {
		t.Errorf("Bob data incorrect: %+v", bob)
	}

	// Get non-existent player
	nobody := game.GetPlayer("Nobody")
	if nobody != nil {
		t.Error("GetPlayer(Nobody) should return nil")
	}
}

// TestRemovePlayer verifies player removal
func TestRemovePlayer(t *testing.T) {
	game := NewBlackjackGame()

	// Add players
	game.AddPlayer("Alice", "alice@test.com", 100)
	game.AddPlayer("Bob", "bob@test.com", 200)
	game.AddPlayer("Charlie", "charlie@test.com", 300)

	if game.PlayerCount() != 3 {
		t.Fatalf("Initial player count = %d; want 3", game.PlayerCount())
	}

	// Remove middle player
	removed := game.RemovePlayer("Bob")
	if !removed {
		t.Error("RemovePlayer(Bob) should return true")
	}

	if game.PlayerCount() != 2 {
		t.Errorf("Player count after removal = %d; want 2", game.PlayerCount())
	}

	// Verify Bob is gone
	bob := game.GetPlayer("Bob")
	if bob != nil {
		t.Error("Bob should be removed")
	}

	// Verify others still there
	if game.GetPlayer("Alice") == nil {
		t.Error("Alice should still be in game")
	}
	if game.GetPlayer("Charlie") == nil {
		t.Error("Charlie should still be in game")
	}

	// Try to remove non-existent player
	removed = game.RemovePlayer("Nobody")
	if removed {
		t.Error("RemovePlayer(Nobody) should return false")
	}
}

// TestGameReset verifies game reset functionality
func TestGameReset(t *testing.T) {
	game := NewBlackjackGame()

	// Add players and set up game state
	game.AddPlayer("Alice", "alice@test.com", 100)
	game.AddPlayer("Bob", "bob@test.com", 200)

	alice := game.GetPlayer("Alice")
	alice.Hand.AddCard(Card{Rank: Ace, Suit: Spade})
	alice.Hand.AddCard(Card{Rank: King, Suit: Heart})
	alice.Status = StatusBlackjack
	alice.IsDoubled = true

	game.Phase = PhaseFinished
	game.Dealer.AddCard(Card{Rank: Queen, Suit: Diamond})
	game.CurrentPlayer = 1

	// Reset game
	game.Reset()

	// Verify reset state
	if game.Phase != PhaseLobby {
		t.Errorf("Phase after reset = %v; want Lobby", game.Phase)
	}

	if game.CurrentPlayer != 0 {
		t.Errorf("CurrentPlayer after reset = %d; want 0", game.CurrentPlayer)
	}

	if game.Dealer.CardCount() != 0 {
		t.Errorf("Dealer hand after reset has %d cards; want 0", game.Dealer.CardCount())
	}

	// Verify players still exist but hands are cleared
	if game.PlayerCount() != 2 {
		t.Errorf("Player count after reset = %d; want 2", game.PlayerCount())
	}

	alice = game.GetPlayer("Alice")
	if alice == nil {
		t.Fatal("Alice should still exist after reset")
	}

	if alice.Hand.CardCount() != 0 {
		t.Errorf("Alice hand after reset has %d cards; want 0", alice.Hand.CardCount())
	}

	if alice.Status != StatusActive {
		t.Errorf("Alice status after reset = %v; want Active", alice.Status)
	}

	if alice.IsDoubled {
		t.Error("Alice IsDoubled should be false after reset")
	}
}

// Feature: blackjack-game, Property 13: Initial card distribution
// TestStartRoundAndDealing verifies card dealing works correctly.
func TestStartRoundAndDealing(t *testing.T) {
	game := NewBlackjackGame()
	game.AddPlayer("Alice", "alice@test.com", 100)
	game.AddPlayer("Bob", "bob@test.com", 200)

	deck := NewDeck()
	deck.SetCards([]Card{
		{Rank: Five, Suit: Spade},   // Alice card 1
		{Rank: Six, Suit: Spade},    // Bob card 1
		{Rank: Seven, Suit: Spade},  // Dealer card 1
		{Rank: Six, Suit: Heart},    // Alice card 2
		{Rank: Seven, Suit: Heart},  // Bob card 2
		{Rank: Eight, Suit: Heart},  // Dealer card 2
	})
	game.Deck = deck

	info, err := game.StartRound()
	if err != nil {
		t.Fatalf("Failed to start round: %v", err)
	}

	if game.Phase != PhasePlayerTurns {
		t.Errorf("Game phase = %v; want PhasePlayerTurns", game.Phase)
	}

	if len(info.Players) != 2 {
		t.Errorf("Info Players length = %d; want 2", len(info.Players))
	}

	// Each player must have exactly 2 cards
	for _, p := range game.Players {
		if p.Hand.CardCount() != 2 {
			t.Errorf("Player %s hand count = %d; want 2", p.Name, p.Hand.CardCount())
		}
	}

	// Dealer must have exactly 2 cards
	if game.Dealer.CardCount() != 2 {
		t.Errorf("Dealer hand count = %d; want 2", game.Dealer.CardCount())
	}

	// Dealer upcard must be the first card
	if game.DealerUpCard != game.Dealer.Cards[0] {
		t.Errorf("DealerUpCard = %v; want first dealer card %v", game.DealerUpCard, game.Dealer.Cards[0])
	}
}

// Feature: blackjack-game, Property 65: Dealer blackjack immediate end
// Feature: blackjack-game, Property 66: Double blackjack push
// TestDealerNaturalBlackjack verifies immediate round end on dealer blackjack.
func TestDealerNaturalBlackjack(t *testing.T) {
	// Case 1: Dealer has Blackjack, player does not
	game1 := NewBlackjackGame()
	game1.AddPlayer("Alice", "alice@test.com", 100)

	// Rig the deck: Player gets (2♠, 3♠), Dealer gets (A♠, K♠)
	riggedDeck1 := NewDeck()
	riggedDeck1.SetCards([]Card{
		{Rank: Two, Suit: Spade},   // Player card 1
		{Rank: Ace, Suit: Spade},   // Dealer card 1
		{Rank: Three, Suit: Spade}, // Player card 2
		{Rank: King, Suit: Spade},  // Dealer card 2
	})
	game1.Deck = riggedDeck1

	_, err := game1.StartRound()
	if err != nil {
		t.Fatalf("Failed to start round: %v", err)
	}

	if game1.Phase != PhaseFinished {
		t.Errorf("Game phase = %v; want PhaseFinished on dealer blackjack", game1.Phase)
	}

	alice := game1.GetPlayer("Alice")
	if alice.Status != StatusStand {
		t.Errorf("Alice status = %v; want StatusStand (lose)", alice.Status)
	}

	// Case 2: Both dealer and player have Blackjack (Push)
	game2 := NewBlackjackGame()
	game2.AddPlayer("Alice", "alice@test.com", 100)

	// Rig the deck: Player gets (A♥, Q♥), Dealer gets (A♠, K♠)
	riggedDeck2 := NewDeck()
	riggedDeck2.SetCards([]Card{
		{Rank: Ace, Suit: Heart},   // Player card 1
		{Rank: Ace, Suit: Spade},   // Dealer card 1
		{Rank: Queen, Suit: Heart}, // Player card 2
		{Rank: King, Suit: Spade},  // Dealer card 2
	})
	game2.Deck = riggedDeck2

	_, err = game2.StartRound()
	if err != nil {
		t.Fatalf("Failed to start round: %v", err)
	}

	if game2.Phase != PhaseFinished {
		t.Errorf("Game phase = %v; want PhaseFinished", game2.Phase)
	}

	alice2 := game2.GetPlayer("Alice")
	if alice2.Status != StatusBlackjack {
		t.Errorf("Alice status = %v; want StatusBlackjack", alice2.Status)
	}

	results := game2.DetermineWinners()
	if len(results) != 1 {
		t.Fatalf("Results length = %d; want 1", len(results))
	}
	if results[0].Outcome != "push" {
		t.Errorf("Outcome = %s; want push", results[0].Outcome)
	}
	if results[0].Payout != 100 {
		t.Errorf("Payout = %d; want 100", results[0].Payout)
	}
}

// Feature: blackjack-game, Property 16: Hit card addition
// Feature: blackjack-game, Property 17: Bust detection on hit
// Feature: blackjack-game, Property 18: Turn advancement on bust
// Feature: blackjack-game, Property 19: Turn persistence on non-bust hit
// TestPlayerActionHit verifies Hit action and transition logic.
func TestPlayerActionHit(t *testing.T) {
	game := NewBlackjackGame()
	game.AddPlayer("Alice", "alice@test.com", 100)
	game.AddPlayer("Bob", "bob@test.com", 200)

	// Rig the deck:
	// Alice: (10♠, 5♠) = 15. Bob: (10♥, 8♥) = 18. Dealer: (10♦, 7♦) = 17.
	// Next cards to draw:
	// - First hit (Alice): 4♠ = 19 (No bust, turn stays)
	// - Second hit (Alice): 8♠ = 27 (Bust, turn advances to Bob)
	riggedDeck := NewDeck()
	riggedDeck.SetCards([]Card{
		{Rank: Ten, Suit: Spade},      // Alice 1 (index 0)
		{Rank: Ten, Suit: Heart},      // Bob 1 (index 1)
		{Rank: Ten, Suit: Diamond},    // Dealer 1 (index 2)
		{Rank: Five, Suit: Spade},     // Alice 2 (index 3)
		{Rank: Eight, Suit: Heart},    // Bob 2 (index 4)
		{Rank: Seven, Suit: Diamond},  // Dealer 2 (index 5)
		{Rank: Four, Suit: Spade},     // Next Card (Alice Hit 1) (index 6)
		{Rank: Eight, Suit: Spade},    // Next Card (Alice Hit 2) (index 7)
	})
	game.Deck = riggedDeck

	_, err := game.StartRound()
	if err != nil {
		t.Fatalf("StartRound failed: %v", err)
	}

	// Verify Alice is current player
	if game.Players[game.CurrentPlayer].Name != "Alice" {
		t.Fatalf("Current player = %s; want Alice", game.Players[game.CurrentPlayer].Name)
	}

	// 1. Alice hits (15 + 4 = 19). Turn remains Alice.
	res, err := game.Hit("Alice")
	if err != nil {
		t.Fatalf("Alice Hit failed: %v", err)
	}
	if !res.Valid {
		t.Error("Action should be valid")
	}
	if res.PlayerBust {
		t.Error("Alice should not be bust")
	}
	if res.NextPlayer != "Alice" {
		t.Errorf("Next player = %s; want Alice", res.NextPlayer)
	}
	alice := game.GetPlayer("Alice")
	if alice.Hand.CardCount() != 3 {
		t.Errorf("Alice hand count = %d; want 3", alice.Hand.CardCount())
	}
	if alice.Hand.Value() != 19 {
		t.Errorf("Alice hand value = %d; want 19", alice.Hand.Value())
	}

	// 2. Alice hits again (19 + 8 = 27). Busts! Turn advances to Bob.
	res, err = game.Hit("Alice")
	if err != nil {
		t.Fatalf("Alice second Hit failed: %v", err)
	}
	if !res.PlayerBust {
		t.Error("Alice should be bust")
	}
	if alice.Status != StatusBust {
		t.Errorf("Alice status = %v; want StatusBust", alice.Status)
	}
	if res.NextPlayer != "Bob" {
		t.Errorf("Next player = %s; want Bob", res.NextPlayer)
	}
	if game.Players[game.CurrentPlayer].Name != "Bob" {
		t.Errorf("Current player index should be Bob, got %s", game.Players[game.CurrentPlayer].Name)
	}
}

// Feature: blackjack-game, Property 21: Stand turn advancement
// Feature: blackjack-game, Property 22: Dealer turn trigger
// Feature: blackjack-game, Property 23: Stand status recording
// TestPlayerActionStand verifies Stand action.
func TestPlayerActionStand(t *testing.T) {
	game := NewBlackjackGame()
	game.AddPlayer("Alice", "alice@test.com", 100)
	game.AddPlayer("Bob", "bob@test.com", 200)

	deck := NewDeck()
	deck.SetCards([]Card{
		{Rank: Five, Suit: Spade},   // Alice card 1
		{Rank: Six, Suit: Spade},    // Bob card 1
		{Rank: Seven, Suit: Spade},  // Dealer card 1
		{Rank: Six, Suit: Heart},    // Alice card 2
		{Rank: Seven, Suit: Heart},  // Bob card 2
		{Rank: Eight, Suit: Heart},  // Dealer card 2
	})
	game.Deck = deck

	_, err := game.StartRound()
	if err != nil {
		t.Fatalf("StartRound failed: %v", err)
	}

	// Alice stands
	res, err := game.Stand("Alice")
	if err != nil {
		t.Fatalf("Alice Stand failed: %v", err)
	}
	if res.NextPlayer != "Bob" {
		t.Errorf("Next player = %s; want Bob", res.NextPlayer)
	}
	alice := game.GetPlayer("Alice")
	if alice.Status != StatusStand {
		t.Errorf("Alice status = %v; want StatusStand", alice.Status)
	}

	// Bob stands -> triggers dealer turn
	res, err = game.Stand("Bob")
	if err != nil {
		t.Fatalf("Bob Stand failed: %v", err)
	}
	if !res.DealerTurn {
		t.Error("Should trigger dealer turn")
	}
	if game.Phase != PhaseDealerTurn {
		t.Errorf("Game phase = %v; want PhaseDealerTurn", game.Phase)
	}
	bob := game.GetPlayer("Bob")
	if bob.Status != StatusStand {
		t.Errorf("Bob status = %v; want StatusStand", bob.Status)
	}
}

// Feature: blackjack-game, Property 26: Double down single card
// Feature: blackjack-game, Property 27: Double down auto-stand
// Feature: blackjack-game, Property 28: Double down card count restriction
// TestPlayerActionDoubleDown verifies Double Down logic.
func TestPlayerActionDoubleDown(t *testing.T) {
	game := NewBlackjackGame()
	game.AddPlayer("Alice", "alice@test.com", 200)
	_ = game.SetPlayerBet("Alice", 100)

	riggedDeck := NewDeck()
	riggedDeck.SetCards([]Card{
		{Rank: Five, Suit: Spade},     // Alice 1
		{Rank: Ten, Suit: Diamond},    // Dealer 1
		{Rank: Six, Suit: Spade},      // Alice 2
		{Rank: Seven, Suit: Diamond},  // Dealer 2
		{Rank: Ten, Suit: Heart},      // Next Card (Double Down card)
	})
	game.Deck = riggedDeck

	_, err := game.StartRound()
	if err != nil {
		t.Fatalf("StartRound failed: %v", err)
	}

	// 1. Valid double down (100 -> 200 bet, receives Ten, stands with 21)
	alice := game.GetPlayer("Alice")
	if alice.Bet != 100 {
		t.Fatalf("Alice initial bet = %d; want 100", alice.Bet)
	}

	res, err := game.DoubleDown("Alice")
	if err != nil {
		t.Fatalf("Double down failed: %v", err)
	}

	if alice.Bet != 200 {
		t.Errorf("Alice bet after double down = %d; want 200", alice.Bet)
	}
	if !alice.IsDoubled {
		t.Error("Alice IsDoubled should be true")
	}
	if alice.Hand.CardCount() != 3 {
		t.Errorf("Alice cards = %d; want 3", alice.Hand.CardCount())
	}
	if alice.Hand.Value() != 21 {
		t.Errorf("Alice hand value = %d; want 21", alice.Hand.Value())
	}
	if alice.Status != StatusStand {
		t.Errorf("Alice status = %v; want StatusStand", alice.Status)
	}
	if !res.DealerTurn {
		t.Error("Double down on sole player should transition to dealer turn")
	}

	// Reset and test invalid double down (card count > 2)
	game.Reset()
	game.AddPlayer("Alice", "alice@test.com", 200)
	riggedDeck2 := NewDeck()
	riggedDeck2.SetCards([]Card{
		{Rank: Five, Suit: Spade},     // Alice 1
		{Rank: Ten, Suit: Diamond},    // Dealer 1
		{Rank: Six, Suit: Spade},      // Alice 2
		{Rank: Seven, Suit: Diamond},  // Dealer 2
		{Rank: Two, Suit: Heart},      // Alice Hit 1 card
		{Rank: Three, Suit: Heart},    // Alice next card (won't be drawn if double fails)
	})
	game.Deck = riggedDeck2
	_, _ = game.StartRound()

	// Alice hits first
	_, _ = game.Hit("Alice")

	// Try to double down with 3 cards (should fail)
	_, err = game.DoubleDown("Alice")
	if err == nil {
		t.Error("Double down with 3 cards should fail")
	}
}

// Feature: blackjack-game, Property 30: Dealer hit rule
// Feature: blackjack-game, Property 31: Dealer stand rule
// Feature: blackjack-game, Property 32: Dealer bust detection
// TestDealerTurnLogic verifies dealer's automatic actions.
func TestDealerTurnLogic(t *testing.T) {
	// Case 1: Dealer stands on 17+
	game1 := NewBlackjackGame()
	game1.AddPlayer("Alice", "alice@test.com", 100)
	
	riggedDeck1 := NewDeck()
	riggedDeck1.SetCards([]Card{
		{Rank: Five, Suit: Spade},      // Alice 1
		{Rank: Ten, Suit: Diamond},     // Dealer 1 (10)
		{Rank: Six, Suit: Spade},       // Alice 2
		{Rank: Seven, Suit: Diamond},   // Dealer 2 (7) -> Total 17 (Stand!)
	})
	game1.Deck = riggedDeck1

	_, _ = game1.StartRound()
	_, _ = game1.Stand("Alice")

	res1 := game1.PlayDealerTurn()
	if res1.FinalValue != 17 {
		t.Errorf("Dealer final value = %d; want 17", res1.FinalValue)
	}
	if len(res1.Cards) != 2 {
		t.Errorf("Dealer card count = %d; want 2 (no extra hits)", len(res1.Cards))
	}
	if res1.IsBust {
		t.Error("Dealer should not be bust")
	}

	// Case 2: Dealer hits on <= 16 and busts
	game2 := NewBlackjackGame()
	game2.AddPlayer("Alice", "alice@test.com", 100)

	riggedDeck2 := NewDeck()
	riggedDeck2.SetCards([]Card{
		{Rank: Five, Suit: Spade},      // Alice 1
		{Rank: Five, Suit: Diamond},    // Dealer 1 (5)
		{Rank: Six, Suit: Spade},       // Alice 2
		{Rank: Six, Suit: Diamond},     // Dealer 2 (6) -> Total 11 (Must hit!)
		{Rank: Five, Suit: Heart},      // Hit 1 -> Total 16 (Must hit!)
		{Rank: Ten, Suit: Club},        // Hit 2 -> Total 26 (Bust!)
	})
	game2.Deck = riggedDeck2

	_, _ = game2.StartRound()
	_, _ = game2.Stand("Alice")

	res2 := game2.PlayDealerTurn()
	if res2.FinalValue != 26 {
		t.Errorf("Dealer final value = %d; want 26", res2.FinalValue)
	}
	if !res2.IsBust {
		t.Error("Dealer should be bust")
	}
}

// Feature: blackjack-game, Property 36: Blackjack payout ratio
// Feature: blackjack-game, Property 37: Regular win payout ratio
// Feature: blackjack-game, Property 38: Push payout
// Feature: blackjack-game, Property 39: Loss payout
// Feature: blackjack-game, Property 40: Dealer bust player win payout
// TestWinnerDetermination verifies outcome and payouts.
func TestWinnerDetermination(t *testing.T) {
	game := NewBlackjackGame()
	
	// Add players
	game.AddPlayer("Alice", "alice@test.com", 100) // Blackjack
	game.AddPlayer("Bob", "bob@test.com", 100)     // Win regular
	game.AddPlayer("Charlie", "charlie@test.com", 100) // Push
	game.AddPlayer("Dave", "dave@test.com", 100)   // Lose
	game.AddPlayer("Eva", "eva@test.com", 100)     // Bust (Lose)

	// Set hands directly to test winner determination logic cleanly
	// Dealer: 18
	game.Dealer.AddCard(Card{Rank: Ten, Suit: Spade})
	game.Dealer.AddCard(Card{Rank: Eight, Suit: Heart})

	// Alice: Blackjack (A + J)
	alice := game.GetPlayer("Alice")
	alice.Hand.AddCard(Card{Rank: Ace, Suit: Spade})
	alice.Hand.AddCard(Card{Rank: Jack, Suit: Heart})

	// Bob: 20
	bob := game.GetPlayer("Bob")
	bob.Hand.AddCard(Card{Rank: Ten, Suit: Heart})
	bob.Hand.AddCard(Card{Rank: King, Suit: Club})

	// Charlie: 18 (Push)
	charlie := game.GetPlayer("Charlie")
	charlie.Hand.AddCard(Card{Rank: Ten, Suit: Diamond})
	charlie.Hand.AddCard(Card{Rank: Eight, Suit: Club})

	// Dave: 17 (Lose)
	dave := game.GetPlayer("Dave")
	dave.Hand.AddCard(Card{Rank: Ten, Suit: Club})
	dave.Hand.AddCard(Card{Rank: Seven, Suit: Club})

	// Eva: Bust
	eva := game.GetPlayer("Eva")
	eva.Hand.AddCard(Card{Rank: Ten, Suit: Spade})
	eva.Hand.AddCard(Card{Rank: Eight, Suit: Heart})
	eva.Hand.AddCard(Card{Rank: Five, Suit: Club}) // 23

	results := game.DetermineWinners()
	resMap := make(map[string]WinResult)
	for _, r := range results {
		resMap[r.PlayerName] = r
	}

	// Verify Alice: Blackjack, 2.5x payout (250)
	rAlice := resMap["Alice"]
	if rAlice.Outcome != "blackjack" || rAlice.Payout != 250 {
		t.Errorf("Alice result: Outcome=%s, Payout=%d; want blackjack, 250", rAlice.Outcome, rAlice.Payout)
	}

	// Verify Bob: Win, 2x payout (200)
	rBob := resMap["Bob"]
	if rBob.Outcome != "win" || rBob.Payout != 200 {
		t.Errorf("Bob result: Outcome=%s, Payout=%d; want win, 200", rBob.Outcome, rBob.Payout)
	}

	// Verify Charlie: Push, 1x payout (100)
	rCharlie := resMap["Charlie"]
	if rCharlie.Outcome != "push" || rCharlie.Payout != 100 {
		t.Errorf("Charlie result: Outcome=%s, Payout=%d; want push, 100", rCharlie.Outcome, rCharlie.Payout)
	}

	// Verify Dave: Lose, 0 payout
	rDave := resMap["Dave"]
	if rDave.Outcome != "lose" || rDave.Payout != 0 {
		t.Errorf("Dave result: Outcome=%s, Payout=%d; want lose, 0", rDave.Outcome, rDave.Payout)
	}

	// Verify Eva: Lose (Bust), 0 payout
	rEva := resMap["Eva"]
	if rEva.Outcome != "lose" || rEva.Payout != 0 {
		t.Errorf("Eva result: Outcome=%s, Payout=%d; want lose, 0", rEva.Outcome, rEva.Payout)
	}
}
