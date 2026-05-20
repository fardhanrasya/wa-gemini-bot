package poker

import (
	"testing"
)

// ==========================================================================
// Hand evaluator tests — memverifikasi bahwa semua 10 jenis hand
// terdeteksi dengan benar, termasuk edge cases dan tiebreakers.
// ==========================================================================

func TestEvaluateBestHand_RoyalFlush(t *testing.T) {
	cards := []Card{
		{Ace, Spade}, {King, Spade}, {Queen, Spade}, {Jack, Spade}, {Ten, Spade},
		{Two, Heart}, {Three, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != RoyalFlush {
		t.Errorf("expected RoyalFlush, got %s", result.Rank)
	}
}

func TestEvaluateBestHand_StraightFlush(t *testing.T) {
	cards := []Card{
		{Nine, Heart}, {Eight, Heart}, {Seven, Heart}, {Six, Heart}, {Five, Heart},
		{Ace, Spade}, {King, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != StraightFlush {
		t.Errorf("expected StraightFlush, got %s", result.Rank)
	}
	if result.TieBreakers[0] != 9 {
		t.Errorf("expected high card 9, got %d", result.TieBreakers[0])
	}
}

func TestEvaluateBestHand_StraightFlush_AceLow(t *testing.T) {
	// Ace-low straight flush (wheel flush): A-2-3-4-5 of same suit
	cards := []Card{
		{Ace, Club}, {Two, Club}, {Three, Club}, {Four, Club}, {Five, Club},
		{King, Heart}, {Queen, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != StraightFlush {
		t.Errorf("expected StraightFlush, got %s", result.Rank)
	}
	// Ace-low straight has high card = 5
	if result.TieBreakers[0] != 5 {
		t.Errorf("expected high card 5 for ace-low straight flush, got %d", result.TieBreakers[0])
	}
}

func TestEvaluateBestHand_FourOfAKind(t *testing.T) {
	cards := []Card{
		{Ace, Spade}, {Ace, Heart}, {Ace, Diamond}, {Ace, Club}, {King, Spade},
		{Two, Heart}, {Three, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != FourOfAKind {
		t.Errorf("expected FourOfAKind, got %s", result.Rank)
	}
	if result.TieBreakers[0] != int(Ace) {
		t.Errorf("expected quads Ace(14), got %d", result.TieBreakers[0])
	}
}

func TestEvaluateBestHand_FullHouse(t *testing.T) {
	cards := []Card{
		{King, Spade}, {King, Heart}, {King, Diamond}, {Four, Club}, {Four, Spade},
		{Two, Heart}, {Three, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != FullHouse {
		t.Errorf("expected FullHouse, got %s", result.Rank)
	}
	if result.TieBreakers[0] != int(King) || result.TieBreakers[1] != int(Four) {
		t.Errorf("expected Kings full of Fours, got trips=%d pair=%d", result.TieBreakers[0], result.TieBreakers[1])
	}
}

func TestEvaluateBestHand_Flush(t *testing.T) {
	cards := []Card{
		{Ace, Heart}, {Jack, Heart}, {Nine, Heart}, {Six, Heart}, {Three, Heart},
		{King, Spade}, {Queen, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != Flush {
		t.Errorf("expected Flush, got %s", result.Rank)
	}
}

func TestEvaluateBestHand_Straight(t *testing.T) {
	cards := []Card{
		{Ten, Spade}, {Nine, Heart}, {Eight, Diamond}, {Seven, Club}, {Six, Spade},
		{Two, Heart}, {Three, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != Straight {
		t.Errorf("expected Straight, got %s", result.Rank)
	}
	if result.TieBreakers[0] != int(Ten) {
		t.Errorf("expected high card 10, got %d", result.TieBreakers[0])
	}
}

func TestEvaluateBestHand_Straight_AceLow(t *testing.T) {
	// Ace-low straight (wheel): A-2-3-4-5
	cards := []Card{
		{Ace, Spade}, {Two, Heart}, {Three, Diamond}, {Four, Club}, {Five, Spade},
		{King, Heart}, {Queen, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != Straight {
		t.Errorf("expected Straight, got %s", result.Rank)
	}
	if result.TieBreakers[0] != 5 {
		t.Errorf("expected high card 5 for ace-low straight, got %d", result.TieBreakers[0])
	}
}

func TestEvaluateBestHand_ThreeOfAKind(t *testing.T) {
	cards := []Card{
		{Eight, Spade}, {Eight, Heart}, {Eight, Diamond}, {Ace, Club}, {King, Spade},
		{Two, Heart}, {Three, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != ThreeOfAKind {
		t.Errorf("expected ThreeOfAKind, got %s", result.Rank)
	}
}

func TestEvaluateBestHand_TwoPair(t *testing.T) {
	cards := []Card{
		{Ace, Spade}, {Ace, Heart}, {Seven, Diamond}, {Seven, Club}, {King, Spade},
		{Two, Heart}, {Three, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != TwoPair {
		t.Errorf("expected TwoPair, got %s", result.Rank)
	}
	if result.TieBreakers[0] != int(Ace) || result.TieBreakers[1] != int(Seven) {
		t.Errorf("expected Aces and Sevens, got %d and %d", result.TieBreakers[0], result.TieBreakers[1])
	}
}

func TestEvaluateBestHand_OnePair(t *testing.T) {
	cards := []Card{
		{Jack, Spade}, {Jack, Heart}, {Ace, Diamond}, {King, Club}, {Nine, Spade},
		{Two, Heart}, {Three, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != OnePair {
		t.Errorf("expected OnePair, got %s", result.Rank)
	}
}

func TestEvaluateBestHand_HighCard(t *testing.T) {
	cards := []Card{
		{Ace, Spade}, {King, Heart}, {Jack, Diamond}, {Nine, Club}, {Seven, Spade},
		{Two, Heart}, {Four, Diamond},
	}
	result := EvaluateBestHand(cards)
	if result.Rank != HighCard {
		t.Errorf("expected HighCard, got %s", result.Rank)
	}
}

// ==========================================================================
// Hand comparison tests
// ==========================================================================

func TestCompareHands_DifferentRanks(t *testing.T) {
	flush := HandResult{Rank: Flush, TieBreakers: []int{14, 11, 9, 6, 3}}
	straight := HandResult{Rank: Straight, TieBreakers: []int{10}}

	if CompareHands(flush, straight) != 1 {
		t.Error("Flush should beat Straight")
	}
	if CompareHands(straight, flush) != -1 {
		t.Error("Straight should lose to Flush")
	}
}

func TestCompareHands_SameRank_DifferentKicker(t *testing.T) {
	// Two pair Aces and Kings with Queen kicker
	hand1 := HandResult{Rank: TwoPair, TieBreakers: []int{14, 13, 12}}
	// Two pair Aces and Kings with Jack kicker
	hand2 := HandResult{Rank: TwoPair, TieBreakers: []int{14, 13, 11}}

	if CompareHands(hand1, hand2) != 1 {
		t.Error("Queen kicker should beat Jack kicker")
	}
}

func TestCompareHands_ExactTie(t *testing.T) {
	hand1 := HandResult{Rank: Flush, TieBreakers: []int{14, 12, 10, 8, 6}}
	hand2 := HandResult{Rank: Flush, TieBreakers: []int{14, 12, 10, 8, 6}}

	if CompareHands(hand1, hand2) != 0 {
		t.Error("Identical hands should tie")
	}
}

// ==========================================================================
// Card rendering tests
// ==========================================================================

func TestRenderCards(t *testing.T) {
	cards := []Card{{Ace, Spade}, {King, Heart}}
	result := RenderCards(cards)
	if result == "" {
		t.Error("RenderCards should not return empty string")
	}
	// Verify it contains expected elements (rank labels and suit names)
	if !containsStr(result, "A") || !containsStr(result, "K") {
		t.Error("RenderCards should contain rank labels")
	}
	if !containsStr(result, "Ace") || !containsStr(result, "King") {
		t.Error("RenderCards for <=2 cards should contain rank names")
	}
}

func TestRenderCards_CommunityCards(t *testing.T) {
	cards := []Card{{Seven, Diamond}, {Jack, Spade}, {Two, Club}}
	result := RenderCards(cards)
	if result == "" {
		t.Error("RenderCards should not return empty string")
	}
	// Community cards (3+) should use compact horizontal format with dots
	if !containsStr(result, "•") {
		t.Error("RenderCards for 3+ cards should use '•' separator")
	}
}

func TestRenderCardsShort(t *testing.T) {
	cards := []Card{{Ace, Spade}, {King, Heart}}
	result := RenderCardsShort(cards)
	if !containsStr(result, "🃏") {
		t.Error("RenderCardsShort should contain 🃏 prefix")
	}
	if !containsStr(result, "A") || !containsStr(result, "K") {
		t.Error("RenderCardsShort should contain rank labels")
	}
}

// ==========================================================================
// Deck tests
// ==========================================================================

func TestNewDeck_Has52Cards(t *testing.T) {
	deck := NewDeck()
	cards := make(map[Card]bool)
	for i := 0; i < 52; i++ {
		c := deck.Draw()
		if cards[c] {
			t.Errorf("duplicate card: %s", c)
		}
		cards[c] = true
	}
	if len(cards) != 52 {
		t.Errorf("expected 52 unique cards, got %d", len(cards))
	}
}

func TestDeck_Shuffle_DifferentOrder(t *testing.T) {
	d1 := NewDeck()
	d2 := NewDeck()

	// Sangat kecil kemungkinannya dua shuffle menghasilkan urutan yang sama
	same := true
	for i := 0; i < 52; i++ {
		if d1.cards[i] != d2.cards[i] {
			same = false
			break
		}
	}
	// Ini tes probabilistik — failure rate ~1/52! ≈ 0
	if same {
		t.Log("Warning: two shuffled decks had the same order (extremely unlikely)")
	}
}

// ==========================================================================
// Pot calculation tests
// ==========================================================================

func TestCalculatePots_SimpleCase(t *testing.T) {
	contributions := []PlayerContribution{
		{Name: "A", Amount: 100, Folded: false},
		{Name: "B", Amount: 100, Folded: false},
		{Name: "C", Amount: 100, Folded: false},
	}
	pots := CalculatePots(contributions)
	if len(pots) != 1 {
		t.Fatalf("expected 1 pot, got %d", len(pots))
	}
	if pots[0].Amount != 300 {
		t.Errorf("expected pot amount 300, got %d", pots[0].Amount)
	}
	if len(pots[0].EligiblePlayers) != 3 {
		t.Errorf("expected 3 eligible players, got %d", len(pots[0].EligiblePlayers))
	}
}

func TestCalculatePots_WithSidePot(t *testing.T) {
	contributions := []PlayerContribution{
		{Name: "A", Amount: 50, Folded: false},  // all-in 50
		{Name: "B", Amount: 100, Folded: false}, // call 100
		{Name: "C", Amount: 100, Folded: false}, // call 100
	}
	pots := CalculatePots(contributions)
	if len(pots) != 2 {
		t.Fatalf("expected 2 pots, got %d", len(pots))
	}
	// Main pot: 50 × 3 = 150 (A, B, C eligible)
	if pots[0].Amount != 150 {
		t.Errorf("expected main pot 150, got %d", pots[0].Amount)
	}
	if len(pots[0].EligiblePlayers) != 3 {
		t.Errorf("expected 3 eligible for main pot, got %d", len(pots[0].EligiblePlayers))
	}
	// Side pot: 50 × 2 = 100 (only B, C eligible)
	if pots[1].Amount != 100 {
		t.Errorf("expected side pot 100, got %d", pots[1].Amount)
	}
	if len(pots[1].EligiblePlayers) != 2 {
		t.Errorf("expected 2 eligible for side pot, got %d", len(pots[1].EligiblePlayers))
	}
}

func TestCalculatePots_WithFoldedPlayer(t *testing.T) {
	contributions := []PlayerContribution{
		{Name: "A", Amount: 50, Folded: true},   // folded but contributed
		{Name: "B", Amount: 100, Folded: false},
		{Name: "C", Amount: 100, Folded: false},
	}
	pots := CalculatePots(contributions)
	// A's chips should go into pot but A is not eligible
	totalPot := 0
	for _, p := range pots {
		totalPot += p.Amount
	}
	if totalPot != 250 {
		t.Errorf("expected total pot 250, got %d", totalPot)
	}
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
