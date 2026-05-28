package trading

import "wa-gemini-bot/internal/economy"

// rankLeverageCaps mendefinisikan batas leverage maksimal berdasarkan pangkat pemain.
// Semakin tinggi pangkat (ditentukan oleh total chip di economy), semakin besar
// leverage yang bisa digunakan — mirip broker real yang memberikan margin lebih
// besar ke klien dengan modal lebih besar.
//
// Cap ini berlaku berdasarkan chip KESELURUHAN (balance economy), bukan saldo trading.
// Ini mencegah pemain baru menggunakan leverage destruktif sebelum memahami risikonya.
var rankLeverageCaps = map[string]int{
	"Peasant":    2,
	"Levy":       3,
	"Mercenary":  5,
	"Governor":   10,
	"Diplomat":   15,
	"General":    20,
	"Strategist": 25,
	"Monarch":    30,
	"Emperor":    40,
	"Hegemon":    50,
}

// MaxLeverageForBalance mengembalikan leverage tertinggi yang diizinkan
// berdasarkan saldo chip keseluruhan pemain di economy service.
func MaxLeverageForBalance(balance int) int {
	rank := economy.GetRankByBalance(balance)
	if cap, ok := rankLeverageCaps[rank.Name]; ok {
		return cap
	}
	return 2 // fallback ke leverage terendah
}
