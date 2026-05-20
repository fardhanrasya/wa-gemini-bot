package poker

import "sort"

// ==========================================================================
// Pot calculation — menghitung main pot dan side pot saat ada all-in.
//
// Side pot terjadi saat seorang player all-in dengan chip lebih sedikit
// dari bet yang sedang berjalan. Player tersebut hanya berhak atas pot
// yang proporsional dengan kontribusinya.
//
// Contoh: Player A all-in 50, Player B raise 100, Player C call 100.
//   - Main Pot: 50 × 3 = 150 (A, B, C eligible)
//   - Side Pot: 50 × 2 = 100 (hanya B dan C eligible)
// ==========================================================================

// Pot merepresentasikan satu pot (main atau side).
type Pot struct {
	Amount          int      // Jumlah chip dalam pot ini
	EligiblePlayers []string // Nama player yang berhak atas pot ini
}

// PlayerContribution merepresentasikan total chip yang dimasukkan
// seorang player ke dalam pot selama satu ronde.
type PlayerContribution struct {
	Name   string
	Amount int
	Folded bool // Player yang fold tidak eligible untuk pot manapun
}

// CalculatePots menghitung main pot dan side pot berdasarkan kontribusi
// masing-masing player. Mengembalikan slice of Pot, diurutkan dari
// main pot ke side pot terakhir.
//
// Algoritma: "peeling" — urutkan kontribusi ascending, lalu untuk setiap
// level kontribusi, "kupas" layer chip dari semua player yang berkontribusi
// setidaknya sebanyak itu.
func CalculatePots(contributions []PlayerContribution) []Pot {
	if len(contributions) == 0 {
		return nil
	}

	// Filter player yang folded — mereka tidak eligible tapi chip-nya tetap masuk pot
	type entry struct {
		name   string
		amount int
		folded bool
	}
	entries := make([]entry, len(contributions))
	for i, c := range contributions {
		entries[i] = entry{name: c.Name, amount: c.Amount, folded: c.Folded}
	}

	// Sort by amount ascending untuk peeling algorithm
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].amount < entries[j].amount
	})

	var pots []Pot
	prevLevel := 0

	for i, e := range entries {
		if e.amount <= prevLevel {
			continue
		}

		// Layer ini = (e.amount - prevLevel) chip dari setiap player yang
		// berkontribusi >= e.amount
		layerSize := e.amount - prevLevel
		var eligible []string
		potAmount := 0

		for j := i; j < len(entries); j++ {
			potAmount += layerSize
			if !entries[j].folded {
				eligible = append(eligible, entries[j].name)
			}
		}

		// Tambahkan juga kontribusi dari player sebelum index i
		// yang punya amount > prevLevel (mereka sudah selesai diproses
		// tapi chip layer ini tetap masuk pot)
		// Sebenarnya, semua player dengan amount >= e.amount berkontribusi
		// layerSize. Player dengan amount < e.amount sudah habis di layer sebelumnya.
		// Jadi loop di atas sudah benar.

		// Hanya buat pot jika ada eligible players
		if len(eligible) > 0 && potAmount > 0 {
			pots = append(pots, Pot{
				Amount:          potAmount,
				EligiblePlayers: eligible,
			})
		} else if potAmount > 0 {
			// Semua player di layer ini sudah fold — chip masuk pot sebelumnya
			if len(pots) > 0 {
				pots[len(pots)-1].Amount += potAmount
			} else {
				// Edge case: semua fold di layer pertama. Ini seharusnya
				// tidak terjadi karena minimal 1 player harus menang.
				pots = append(pots, Pot{
					Amount:          potAmount,
					EligiblePlayers: nil,
				})
			}
		}

		prevLevel = e.amount
	}

	return pots
}
