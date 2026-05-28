package trading

// TutorialProgress melacak kemajuan tutorial pemain.
// Tutorial mengajarkan dasar-dasar trading sebelum pemain
// menggunakan saldo real, mencegah kerugian karena ketidaktahuan.
type TutorialProgress struct {
	JID            string `json:"jid"`
	CompletedSteps int    `json:"completed_steps"` // 0-7
	PracticeDone   bool   `json:"practice_done"`
}

// TutorialStep berisi konten satu langkah tutorial.
type TutorialStep struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageHint   string `json:"image_hint"` // hint untuk UI tentang apa yang ditampilkan
}

// TutorialSteps mendefinisikan 7 langkah tutorial yang harus diselesaikan pemain.
// Setiap langkah mengajarkan satu konsep penting dalam trading.
var TutorialSteps = []TutorialStep{
	{
		ID:    1,
		Title: "📊 Membaca Candlestick",
		Description: `Setiap batang (candle) mewakili pergerakan harga dalam 1 periode:

🟢 *Candle Hijau*: Harga naik (Open < Close)
🔴 *Candle Merah*: Harga turun (Open > Close)

Bagian tebal (body) menunjukkan range Open-Close.
Garis tipis (wick/shadow) menunjukkan High dan Low.

Wick panjang ke bawah = tekanan beli kuat.
Wick panjang ke atas = tekanan jual kuat.`,
		ImageHint: "candlestick_basics",
	},
	{
		ID:    2,
		Title: "📈 Mengenali Pattern Chart",
		Description: `Chart tidak bergerak random — ada pola yang berulang. Contoh:

*Double Bottom* (bentuk W): Harga turun 2x ke level yang sama lalu naik → sinyal BULLISH
*Double Top* (bentuk M): Harga naik 2x ke level yang sama lalu turun → sinyal BEARISH
*Bull Flag*: Rally tajam → koreksi kecil sideways → lanjut naik
*Bear Flag*: Drop tajam → bounce kecil → lanjut turun

Pemain yang bisa mengenali pattern ini punya keuntungan besar!
⚠️ Pattern tidak selalu berhasil — kadang gagal (false breakout).`,
		ImageHint: "pattern_examples",
	},
	{
		ID:    3,
		Title: "📉 Indikator Teknikal",
		Description: `Indikator membantu mengkonfirmasi pattern:

*MA20* (garis biru): Rata-rata harga 20 candle terakhir.
- Harga di atas MA = trend naik
- Harga di bawah MA = trend turun
- Harga cross MA = sinyal perubahan trend

*RSI*: Mengukur momentum (0-100).
- RSI > 70 = overbought (mungkin akan turun)
- RSI < 30 = oversold (mungkin akan naik)

*Volume*: Banyaknya transaksi. Volume tinggi saat breakout = sinyal kuat.`,
		ImageHint: "indicator_guide",
	},
	{
		ID:    4,
		Title: "🔄 LONG vs SHORT",
		Description: `Ada 2 cara profit di trading:

📈 *LONG* (Beli): Kamu profit jika harga NAIK setelah beli.
Contoh: Beli di 100, jual di 120 → profit 20%

📉 *SHORT* (Jual): Kamu profit jika harga TURUN setelah jual.
Contoh: Short di 100, close di 80 → profit 20%

Pilih LONG jika kamu melihat sinyal bullish (harga akan naik).
Pilih SHORT jika kamu melihat sinyal bearish (harga akan turun).`,
		ImageHint: "long_short_explain",
	},
	{
		ID:    5,
		Title: "⚡ Leverage",
		Description: `Leverage memperbesar potensi profit DAN loss:

1x leverage: Harga naik 10% → profit 10%
5x leverage: Harga naik 10% → profit 50%
10x leverage: Harga naik 10% → profit 100%

⚠️ TAPI juga sebaliknya:
10x leverage: Harga turun 10% → LOSS 100% (LIKUIDASI!)

Leverage tinggi = high risk, high reward.
Pemula disarankan mulai dari 1x-2x.

Leverage maksimal kamu ditentukan oleh rank chip keseluruhan.`,
		ImageHint: "leverage_explain",
	},
	{
		ID:    6,
		Title: "🛡️ Stop Loss & Take Profit",
		Description: `Risk management adalah kunci trading sukses:

*Stop Loss (SL)*: Harga di mana posisi otomatis ditutup untuk membatasi kerugian.
Contoh: Beli di 100, set SL di 95 → maksimal rugi 5%

*Take Profit (TP)*: Harga di mana posisi otomatis ditutup untuk mengamankan profit.
Contoh: Beli di 100, set TP di 115 → otomatis profit 15%

*Trailing Stop*: SL yang bergerak mengikuti profit. Jika harga naik, SL naik juga.
Contoh: Trailing 3% → jika harga naik ke 120, SL = 116.4

🔑 Aturan emas: Selalu set Stop Loss sebelum buka posisi!`,
		ImageHint: "risk_management",
	},
	{
		ID:    7,
		Title: "📰 Membaca Berita",
		Description: `Berita memberikan konteks fundamental — tapi HATI-HATI:

✅ *Berita Genuine*: Selaras dengan data chart. Contoh: revenue naik + chart bullish = konfirmasi kuat.

⚠️ *Berita Trap*: Terdengar positif tapi sebenarnya misleading. 
Contoh: "Analis bilang saatnya beli!" — tapi chartnya menunjukkan distribusi.

🔇 *Berita Neutral*: Tidak mempengaruhi harga. Abaikan saja.

💡 Tips: Jangan pernah trade hanya berdasarkan berita! Selalu gabungkan dengan analisa chart dan indikator.`,
		ImageHint: "news_analysis",
	},
}

// IsTutorialComplete mengecek apakah pemain sudah menyelesaikan semua tutorial.
func IsTutorialComplete(progress *TutorialProgress) bool {
	return progress.CompletedSteps >= len(TutorialSteps) && progress.PracticeDone
}
