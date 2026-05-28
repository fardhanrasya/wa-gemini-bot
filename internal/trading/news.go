package trading

import "math/rand"

// NewsEvent merepresentasikan satu berita yang muncul selama fase observasi chart.
// Pemain yang terampil membaca berita + chart bersama untuk membuat keputusan,
// sementara pemain yang hanya mengandalkan berita bisa terjebak oleh trap headlines.
type NewsEvent struct {
	Time      int       `json:"time"`      // muncul di candle ke-N
	Headline  string    `json:"headline"`
	Sentiment Sentiment `json:"sentiment"` // sentimen sebenarnya (hidden dari client)
	Impact    string    `json:"impact"`    // "high", "medium", "low"
}

// Sentiment merepresentasikan arah sentimen sebenarnya dari sebuah berita.
type Sentiment string

const (
	SentimentBullish Sentiment = "bullish"
	SentimentBearish Sentiment = "bearish"
	SentimentNeutral Sentiment = "neutral"
	SentimentTrap    Sentiment = "trap" // terlihat satu arah tapi sebenarnya sebaliknya
)

// ─────────────────────────────────────────────────────────────
// NEWS TEMPLATE POOLS
// Setiap pool berisi headline yang bervariasi agar tidak repetitif.
// Template menggunakan placeholder %s untuk nama aset fiktif.
// ─────────────────────────────────────────────────────────────

var bullishNews = []string{
	"📰 Investor institusi besar mulai akumulasi agresif di level support saat ini",
	"📰 Revenue Q2 melampaui ekspektasi analis — pertumbuhan 34% YoY",
	"📰 Partnership strategis dengan konglomerat Fortune 500 resmi diumumkan",
	"📰 Regulator memberikan persetujuan penuh untuk ekspansi ke pasar Asia",
	"📰 Laporan internal menunjukkan backlog pesanan meningkat 200%",
	"📰 Hedge fund ternama mengumumkan posisi long besar dalam filing terbaru",
	"📰 Patent teknologi baru disetujui — competitive advantage diperkuat",
	"📰 Dividen kuartal ini dinaikkan 25 persen sebagai tanda kepercayaan manajemen",
	"📰 Upgrade rating dari 3 analis besar secara bersamaan dalam seminggu terakhir",
	"📰 Insider buying meningkat tajam — 5 eksekutif beli saham pekan ini",
}

var bearishNews = []string{
	"📰 CFO perusahaan mendadak mengundurkan diri tanpa penjelasan resmi",
	"📰 Investigasi regulator dimulai terhadap dugaan manipulasi laporan keuangan",
	"📰 Kompetitor meluncurkan produk generasi baru dengan harga 40% lebih murah",
	"📰 Supply chain terganggu parah — estimasi kerugian mencapai $2 miliar",
	"📰 Gugatan class action diajukan oleh pemegang saham atas dugaan fraud",
	"📰 Downgrade rating dari Goldman Sachs — target price dipangkas 35%",
	"📰 Insider selling masif: CEO menjual 80% kepemilikan saham pribadinya",
	"📰 Produk utama dilarang di 3 negara besar karena masalah regulasi",
	"📰 Debt-to-equity ratio melonjak — risiko default meningkat signifikan",
	"📰 Customer churn rate naik 150% — loyalitas pasar terkikis parah",
}

// trapNews terlihat positif tapi sebenarnya negatif, atau sebaliknya.
// Pemain yang hanya baca headline tanpa analisa chart akan terjebak.
var trapNews = []string{
	"📰 Analis terkenal: 'Ini titik bottom, saatnya all-in beli!'",
	"📰 Short seller terbesar menarik posisinya — apakah ini sinyal reversal?",
	"📰 Volume pembelian melonjak 300% — FOMO atau genuine demand?",
	"📰 Manajemen menyatakan optimis di tengah penurunan revenue 3 kuartal berturut-turut",
	"📰 'Rally ini sangat sehat dan berkelanjutan' kata CEO yang sebelumnya salah prediksi 4 kali",
	"📰 Harga mencapai all-time high — analis memprediksi kenaikan 50% lagi",
	"📰 Perusahaan mengumumkan stock buyback besar di tengah arus kas negatif",
	"📰 Akuisisi perusahaan kecil diumumkan — pasar merespons dengan euforia",
	"📰 'Fundamentalnya kuat' kata analis yang sama yang merekomendasikan Enron tahun 2001",
	"📰 Breakout dari resistance — tapi volume tipis dan RSI sudah overbought",
}

var neutralNews = []string{
	"📰 Perusahaan merilis laporan CSR tahunan — tidak ada dampak material",
	"📰 CEO menghadiri konferensi teknologi di Singapore sebagai keynote speaker",
	"📰 Kantor cabang baru dibuka di lokasi strategis — dampak jangka panjang",
	"📰 Rotasi anggota dewan direksi — perubahan rutin sesuai jadwal",
	"📰 Tim R&D mempresentasikan prototipe terbaru di pameran industri",
	"📰 Perusahaan merayakan ulang tahun ke-25 dengan kampanye marketing baru",
}

// generateNewsEvents menghasilkan 2-3 berita untuk satu sesi trading.
//
// Komposisi berita dirancang agar tidak trivial:
//   - Satu berita GENUINE yang selaras dengan arah pattern (membantu pemain terampil)
//   - Satu berita TRAP atau NEUTRAL (menguji kemampuan filtering informasi)
//   - Opsional: satu berita tambahan (bisa genuine, trap, atau neutral)
//
// expectedDir menentukan arah pattern yang sebenarnya (bullish/bearish),
// agar berita genuine bisa selaras dan berita trap berlawanan.
func generateNewsEvents(expectedDir Direction, rng *rand.Rand) []NewsEvent {
	events := make([]NewsEvent, 0, 3)

	// Berita 1: Genuine — selaras dengan arah pattern (muncul di awal)
	var genuinePool []string
	if expectedDir == DirBullish {
		genuinePool = bullishNews
	} else {
		genuinePool = bearishNews
	}
	events = append(events, NewsEvent{
		Time:      8 + rng.Intn(5), // candle ke-8 sampai ke-12
		Headline:  genuinePool[rng.Intn(len(genuinePool))],
		Sentiment: Sentiment(expectedDir),
		Impact:    "high",
	})

	// Berita 2: Trap atau Neutral (muncul di tengah — menguji kemampuan analisa)
	if rng.Float64() < 0.6 { // 60% chance trap, 40% neutral
		events = append(events, NewsEvent{
			Time:      18 + rng.Intn(5), // candle ke-18 sampai ke-22
			Headline:  trapNews[rng.Intn(len(trapNews))],
			Sentiment: SentimentTrap,
			Impact:    "medium",
		})
	} else {
		events = append(events, NewsEvent{
			Time:      18 + rng.Intn(5),
			Headline:  neutralNews[rng.Intn(len(neutralNews))],
			Sentiment: SentimentNeutral,
			Impact:    "low",
		})
	}

	// Berita 3 (opsional, 50% chance): tambahan untuk variasi
	if rng.Float64() < 0.5 {
		pool := neutralNews
		sentiment := SentimentNeutral
		impact := "low"

		roll := rng.Float64()
		if roll < 0.3 {
			// 30%: berita genuine kedua (reinforcing signal)
			pool = genuinePool
			sentiment = Sentiment(expectedDir)
			impact = "medium"
		} else if roll < 0.5 {
			// 20%: berita trap kedua
			pool = trapNews
			sentiment = SentimentTrap
			impact = "medium"
		}
		// 50%: tetap neutral (default)

		events = append(events, NewsEvent{
			Time:      28 + rng.Intn(5), // candle ke-28 sampai ke-32
			Headline:  pool[rng.Intn(len(pool))],
			Sentiment: sentiment,
			Impact:    impact,
		})
	}

	return events
}
