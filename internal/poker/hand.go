package poker

import "sort"

// ==========================================================================
// Hand evaluation — mengevaluasi 7 kartu menjadi kombinasi 5 kartu terbaik.
//
// Ini adalah bagian paling algoritmis dari engine. Kita mengevaluasi semua
// C(7,5) = 21 kombinasi dari 7 kartu (2 hole + 5 community) untuk mencari
// hand terbaik. Pendekatan brute-force ini simpel dan benar — performance
// bukan masalah karena evaluasi hanya terjadi saat showdown (max 8 kali).
//
// HandRank disimpan sebagai integer sehingga perbandingan cukup dengan >/</==.
// Kicker cards disimpan terurut descending untuk tiebreaker.
// ==========================================================================

// HandRank merepresentasikan peringkat kombinasi kartu poker.
// Nilai lebih tinggi = kombinasi lebih kuat.
type HandRank int

const (
	HighCard      HandRank = iota // Kartu tertinggi
	OnePair                       // Sepasang
	TwoPair                       // Dua pasang
	ThreeOfAKind                  // Three of a Kind
	Straight                      // Straight (berurutan)
	Flush                         // Flush (satu jenis)
	FullHouse                     // Full House
	FourOfAKind                   // Four of a Kind
	StraightFlush                 // Straight Flush
	RoyalFlush                    // Royal Flush
)

var handRankNames = map[HandRank]string{
	HighCard:      "High Card",
	OnePair:       "One Pair",
	TwoPair:       "Two Pair",
	ThreeOfAKind:  "Three of a Kind",
	Straight:      "Straight",
	Flush:         "Flush",
	FullHouse:     "Full House",
	FourOfAKind:   "Four of a Kind",
	StraightFlush: "Straight Flush",
	RoyalFlush:    "Royal Flush",
}

func (h HandRank) String() string { return handRankNames[h] }

// HandResult adalah hasil evaluasi satu hand 5-kartu.
// Rank menentukan kategori (pair, flush, dsb.), TieBreakers menentukan
// pemenang jika dua hand punya Rank yang sama.
//
// TieBreakers berisi nilai-nilai signifikan terurut descending.
// Contoh: Full House (K,K,K,4,4) → TieBreakers = [13, 4]
//         Two Pair (A,A,7,7,3)   → TieBreakers = [14, 7, 3]
type HandResult struct {
	Rank        HandRank
	TieBreakers []int  // Nilai signifikan untuk tiebreaker, descending
	BestCards   []Card // 5 kartu terbaik yang membentuk hand ini
	Description string // Human-readable, misal "Full House (Kings full of Fours)"
}

// CompareHands membandingkan dua HandResult.
// Return: 1 jika a menang, -1 jika b menang, 0 jika seri.
func CompareHands(a, b HandResult) int {
	if a.Rank > b.Rank {
		return 1
	}
	if a.Rank < b.Rank {
		return -1
	}

	// Rank sama — bandingkan tiebreakers satu per satu
	for i := 0; i < len(a.TieBreakers) && i < len(b.TieBreakers); i++ {
		if a.TieBreakers[i] > b.TieBreakers[i] {
			return 1
		}
		if a.TieBreakers[i] < b.TieBreakers[i] {
			return -1
		}
	}

	return 0 // Benar-benar seri (split pot)
}

// EvaluateBestHand mengevaluasi 7 kartu dan mengembalikan kombinasi
// 5 kartu terbaik. Ini adalah entry point utama — caller hanya perlu
// memasukkan 7 kartu, semua kompleksitas evaluasi tersembunyi.
func EvaluateBestHand(cards []Card) HandResult {
	if len(cards) < 5 {
		// Seharusnya tidak terjadi, tapi handle gracefully
		return evaluateFiveCards(cards)
	}

	var best HandResult
	first := true

	// Evaluasi semua kombinasi C(n,5) dari kartu yang tersedia
	combos := combinations(cards, 5)
	for _, combo := range combos {
		result := evaluateFiveCards(combo)
		if first || CompareHands(result, best) > 0 {
			best = result
			first = false
		}
	}

	return best
}

// ==========================================================================
// Evaluasi 5 kartu — core logic
// ==========================================================================

// evaluateFiveCards mengevaluasi tepat 5 kartu menjadi HandResult.
// Urutan pengecekan dari tertinggi ke terendah — return di hit pertama
// agar kode flat dan mudah dibaca (guard clause pattern).
func evaluateFiveCards(cards []Card) HandResult {
	sorted := make([]Card, len(cards))
	copy(sorted, cards)
	sortByRankDesc(sorted)

	isFlush := checkFlush(sorted)
	straight, highCard := checkStraight(sorted)

	// Royal Flush: straight flush dengan high card Ace
	if isFlush && straight && highCard == int(Ace) {
		return HandResult{
			Rank:        RoyalFlush,
			TieBreakers: []int{int(Ace)},
			BestCards:   sorted,
			Description: "Royal Flush",
		}
	}

	// Straight Flush
	if isFlush && straight {
		return HandResult{
			Rank:        StraightFlush,
			TieBreakers: []int{highCard},
			BestCards:   sorted,
			Description: formatStraightFlush(highCard),
		}
	}

	// Four of a Kind
	if quads, kicker, ok := checkFourOfAKind(sorted); ok {
		return HandResult{
			Rank:        FourOfAKind,
			TieBreakers: []int{quads, kicker},
			BestCards:   sorted,
			Description: formatFourOfAKind(quads),
		}
	}

	// Full House
	if trips, pair, ok := checkFullHouse(sorted); ok {
		return HandResult{
			Rank:        FullHouse,
			TieBreakers: []int{trips, pair},
			BestCards:   sorted,
			Description: formatFullHouse(trips, pair),
		}
	}

	// Flush
	if isFlush {
		tieBreakers := rankValues(sorted)
		return HandResult{
			Rank:        Flush,
			TieBreakers: tieBreakers,
			BestCards:   sorted,
			Description: "Flush",
		}
	}

	// Straight
	if straight {
		return HandResult{
			Rank:        Straight,
			TieBreakers: []int{highCard},
			BestCards:   sorted,
			Description: formatStraight(highCard),
		}
	}

	// Three of a Kind
	if trips, kickers, ok := checkThreeOfAKind(sorted); ok {
		tb := append([]int{trips}, kickers...)
		return HandResult{
			Rank:        ThreeOfAKind,
			TieBreakers: tb,
			BestCards:   sorted,
			Description: formatThreeOfAKind(trips),
		}
	}

	// Two Pair
	if highPair, lowPair, kicker, ok := checkTwoPair(sorted); ok {
		return HandResult{
			Rank:        TwoPair,
			TieBreakers: []int{highPair, lowPair, kicker},
			BestCards:   sorted,
			Description: formatTwoPair(highPair, lowPair),
		}
	}

	// One Pair
	if pair, kickers, ok := checkOnePair(sorted); ok {
		tb := append([]int{pair}, kickers...)
		return HandResult{
			Rank:        OnePair,
			TieBreakers: tb,
			BestCards:   sorted,
			Description: formatOnePair(pair),
		}
	}

	// High Card
	tieBreakers := rankValues(sorted)
	return HandResult{
		Rank:        HighCard,
		TieBreakers: tieBreakers,
		BestCards:   sorted,
		Description: formatHighCard(int(sorted[0].Rank)),
	}
}

// ==========================================================================
// Check functions — setiap fungsi mengecek satu jenis kombinasi.
// Return values termasuk info untuk tiebreaker.
// ==========================================================================

func checkFlush(cards []Card) bool {
	suit := cards[0].Suit
	for _, c := range cards[1:] {
		if c.Suit != suit {
			return false
		}
	}
	return true
}

// checkStraight mengecek apakah 5 kartu membentuk straight.
// Menangani kasus khusus Ace-low straight (A-2-3-4-5 = "wheel").
// Return: (isStraight, highCard).
func checkStraight(cards []Card) (bool, int) {
	ranks := rankValues(cards)

	// Normal straight: kartu berurutan turun
	isStraight := true
	for i := 0; i < len(ranks)-1; i++ {
		if ranks[i]-ranks[i+1] != 1 {
			isStraight = false
			break
		}
	}
	if isStraight {
		return true, ranks[0]
	}

	// Ace-low straight (wheel): A-5-4-3-2
	// ranks sudah sorted desc, jadi [14, 5, 4, 3, 2]
	if ranks[0] == int(Ace) && ranks[1] == 5 && ranks[2] == 4 && ranks[3] == 3 && ranks[4] == 2 {
		return true, 5 // High card = 5, bukan Ace (ini straight terendah)
	}

	return false, 0
}

func checkFourOfAKind(cards []Card) (quads, kicker int, ok bool) {
	groups := groupByRank(cards)
	for rank, count := range groups {
		if count == 4 {
			// Cari kicker (kartu yang bukan quads)
			for _, c := range cards {
				if int(c.Rank) != rank {
					return rank, int(c.Rank), true
				}
			}
		}
	}
	return 0, 0, false
}

func checkFullHouse(cards []Card) (trips, pair int, ok bool) {
	groups := groupByRank(cards)
	tripsRank, pairRank := 0, 0
	for rank, count := range groups {
		if count == 3 {
			tripsRank = rank
		} else if count == 2 {
			pairRank = rank
		}
	}
	if tripsRank > 0 && pairRank > 0 {
		return tripsRank, pairRank, true
	}
	return 0, 0, false
}

func checkThreeOfAKind(cards []Card) (trips int, kickers []int, ok bool) {
	groups := groupByRank(cards)
	tripsRank := 0
	for rank, count := range groups {
		if count == 3 {
			tripsRank = rank
		}
	}
	if tripsRank == 0 {
		return 0, nil, false
	}

	var kicks []int
	for _, c := range cards {
		if int(c.Rank) != tripsRank {
			kicks = append(kicks, int(c.Rank))
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(kicks)))
	return tripsRank, kicks, true
}

func checkTwoPair(cards []Card) (highPair, lowPair, kicker int, ok bool) {
	groups := groupByRank(cards)
	var pairs []int
	kickerRank := 0
	for rank, count := range groups {
		if count == 2 {
			pairs = append(pairs, rank)
		} else {
			kickerRank = rank
		}
	}
	if len(pairs) != 2 {
		return 0, 0, 0, false
	}
	sort.Sort(sort.Reverse(sort.IntSlice(pairs)))
	return pairs[0], pairs[1], kickerRank, true
}

func checkOnePair(cards []Card) (pair int, kickers []int, ok bool) {
	groups := groupByRank(cards)
	pairRank := 0
	for rank, count := range groups {
		if count == 2 {
			pairRank = rank
		}
	}
	if pairRank == 0 {
		return 0, nil, false
	}

	var kicks []int
	for _, c := range cards {
		if int(c.Rank) != pairRank {
			kicks = append(kicks, int(c.Rank))
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(kicks)))
	return pairRank, kicks, true
}

// ==========================================================================
// Utilities
// ==========================================================================

// sortByRankDesc mengurutkan kartu dari rank tertinggi ke terendah.
func sortByRankDesc(cards []Card) {
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].Rank > cards[j].Rank
	})
}

// rankValues mengembalikan slice of int rank values, terurut descending.
func rankValues(cards []Card) []int {
	vals := make([]int, len(cards))
	for i, c := range cards {
		vals[i] = int(c.Rank)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(vals)))
	return vals
}

// groupByRank menghitung jumlah kartu per rank.
func groupByRank(cards []Card) map[int]int {
	groups := make(map[int]int)
	for _, c := range cards {
		groups[int(c.Rank)]++
	}
	return groups
}

// combinations mengembalikan semua kombinasi k elemen dari slice.
func combinations(cards []Card, k int) [][]Card {
	var result [][]Card
	var combo []Card
	var generate func(start int)
	generate = func(start int) {
		if len(combo) == k {
			c := make([]Card, k)
			copy(c, combo)
			result = append(result, c)
			return
		}
		for i := start; i < len(cards); i++ {
			combo = append(combo, cards[i])
			generate(i + 1)
			combo = combo[:len(combo)-1]
		}
	}
	generate(0)
	return result
}

// ==========================================================================
// Description formatters — human-readable descriptions for hand ranks.
// ==========================================================================

func rankLabel(r int) string {
	return Rank(r).String()
}

func formatStraightFlush(highCard int) string {
	return "Straight Flush (" + rankLabel(highCard) + "-high)"
}

func formatFourOfAKind(quads int) string {
	return "Four of a Kind (" + rankLabel(quads) + "s)"
}

func formatFullHouse(trips, pair int) string {
	return "Full House (" + rankLabel(trips) + "s full of " + rankLabel(pair) + "s)"
}

func formatStraight(highCard int) string {
	return "Straight (" + rankLabel(highCard) + "-high)"
}

func formatThreeOfAKind(trips int) string {
	return "Three of a Kind (" + rankLabel(trips) + "s)"
}

func formatTwoPair(high, low int) string {
	return "Two Pair (" + rankLabel(high) + "s and " + rankLabel(low) + "s)"
}

func formatOnePair(pair int) string {
	return "One Pair (" + rankLabel(pair) + "s)"
}

func formatHighCard(high int) string {
	return "High Card (" + rankLabel(high) + ")"
}
