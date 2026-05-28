package trading

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// ─────────────────────────────────────────────────────────────
// CORE TYPES
// ─────────────────────────────────────────────────────────────

// Direction merepresentasikan arah pergerakan harga yang diharapkan pattern.
type Direction string

const (
	DirBullish Direction = "long"
	DirBearish Direction = "short"
)

// Scan implements the sql.Scanner interface for Direction, converting legacy "bullish"/"bearish" values to "long"/"short".
func (d *Direction) Scan(value interface{}) error {
	if value == nil {
		*d = ""
		return nil
	}
	var strVal string
	switch v := value.(type) {
	case string:
		strVal = v
	case []byte:
		strVal = string(v)
	default:
		return fmt.Errorf("invalid type for Direction: %T", value)
	}

	switch strVal {
	case "bullish", "long":
		*d = DirBullish
	case "bearish", "short":
		*d = DirBearish
	default:
		*d = Direction(strVal)
	}
	return nil
}

// PricePoint merepresentasikan satu candlestick OHLCV pada chart.
type PricePoint struct {
	Time   int     `json:"t"` // candle ke-N (1 candle = 1 detik)
	Open   float64 `json:"o"`
	High   float64 `json:"h"`
	Low    float64 `json:"l"`
	Close  float64 `json:"c"`
	Volume float64 `json:"v"`
}

// ChartIndicators berisi data indikator teknikal yang dihitung dari price data.
type ChartIndicators struct {
	MA20    []float64 `json:"ma20"`     // Moving Average 20 periods
	RSI     []float64 `json:"rsi"`      // RSI 14 periods
	MACD    []float64 `json:"macd"`     // MACD line (EMA12 - EMA26)
	MACDSig []float64 `json:"macd_sig"` // MACD signal (EMA9 of MACD)
}

// ChartSession merepresentasikan satu sesi chart lengkap yang sudah di-generate.
// Observation dikirim ke client saat sesi dimulai. Resolution dikirim setelah
// fase observasi berakhir — pemain tidak bisa melihat resolution sebelum waktunya.
type ChartSession struct {
	Seed            int64           `json:"seed"`
	PatternType     string          `json:"pattern_type"`
	PatternName     string          `json:"pattern_name"`
	Difficulty      string          `json:"difficulty"`
	ExpectedDir     Direction       `json:"expected_dir"`
	WillResolve     bool            `json:"-"` // hidden dari client
	NoiseLevel      float64         `json:"-"` // hidden dari client
	ObservationData []PricePoint    `json:"observation"`
	ResolutionData  []PricePoint    `json:"-"` // hidden sampai waktunya
	NewsEvents      []NewsEvent     `json:"news"`
	Indicators      ChartIndicators `json:"indicators"`
}

// ─────────────────────────────────────────────────────────────
// DIFFICULTY PARAMETERS
// Disesuaikan agar mendekati kesulitan trading real:
//   - Pattern reliability 48-52% agar pattern tidak bisa dipakai sebagai sinyal auto-win
//   - Noise ratio 108-180% untuk membuat entry timing lebih sulit dan chart lebih choppy
// ─────────────────────────────────────────────────────────────

const (
	patternReliabilityMin = 0.50
	patternReliabilityMax = 0.55
	noiseRatioMin         = 1.08
	noiseRatioMax         = 1.80

	observationCandles = 40 // fase observasi: 40 candle (40 detik)
	resolutionCandles  = 20 // fase resolusi: 20 candle (20 detik)
	totalCandles       = observationCandles + resolutionCandles
)

// ─────────────────────────────────────────────────────────────
// CHART GENERATION
// ─────────────────────────────────────────────────────────────

// GenerateSession membuat sesi chart baru berdasarkan seed deterministik.
//
// Algoritma:
//  1. Derive seed dari JID + timestamp (unik per pemain per waktu)
//  2. Pilih pattern dari pool (weighted random berdasarkan difficulty)
//  3. Tentukan noise level (random 108-180%)
//  4. Tentukan apakah pattern resolve benar (48-52% chance yes)
//  5. Generate price data (observation + resolution)
//  6. Generate news events
//  7. Hitung indikator teknikal
func GenerateSession(jid string) *ChartSession {
	seed := deriveSeed(jid)
	rng := rand.New(rand.NewSource(seed))

	// 1. Pilih pattern
	pattern := selectPattern(rng)

	// 2. Tentukan noise level
	noiseLevel := noiseRatioMin + rng.Float64()*(noiseRatioMax-noiseRatioMin)

	// 3. Tentukan apakah pattern resolve sesuai ekspektasi
	reliability := patternReliabilityMin + rng.Float64()*(patternReliabilityMax-patternReliabilityMin)
	willResolve := rng.Float64() < reliability

	// 4. Generate base price (range 50-500, terasa seperti saham/crypto real)
	basePrice := 50.0 + rng.Float64()*450.0

	// 5. Build price data
	allCandles := pattern.Build(basePrice, noiseLevel, willResolve, rng)

	// Pisahkan observation vs resolution
	observation := allCandles[:observationCandles]
	resolution := allCandles[observationCandles:]

	// 6. Generate news events
	expectedDir := pattern.ExpectedDirection()
	if !willResolve {
		// Jika pattern gagal, direction terbalik
		if expectedDir == DirBullish {
			expectedDir = DirBearish
		} else {
			expectedDir = DirBullish
		}
	}
	news := generateNewsEvents(pattern.ExpectedDirection(), rng)

	// 7. Hitung indikator dari seluruh data (client hanya terima observation indicators)
	indicators := calculateIndicators(allCandles)
	observationIndicators := truncateIndicators(indicators, observationCandles)

	return &ChartSession{
		Seed:            seed,
		PatternType:     pattern.Type(),
		PatternName:     pattern.Name(),
		Difficulty:      pattern.Difficulty(),
		ExpectedDir:     pattern.ExpectedDirection(),
		WillResolve:     willResolve,
		NoiseLevel:      noiseLevel,
		ObservationData: observation,
		ResolutionData:  resolution,
		NewsEvents:      news,
		Indicators:      observationIndicators,
	}
}

// GeneratePracticeSession membuat sesi latihan tutorial dengan noise minimal
// dan pattern yang mudah dikenali. Sesi ini menggunakan saldo virtual.
func GeneratePracticeSession(jid string) *ChartSession {
	seed := deriveSeed(jid)
	rng := rand.New(rand.NewSource(seed))

	// Pilih pattern mudah (Easy difficulty)
	easyPatterns := []Pattern{
		&DoubleBottomPattern{},
		&DoubleTopPattern{},
		&BullFlagPattern{},
		&BearFlagPattern{},
	}
	pattern := easyPatterns[rng.Intn(len(easyPatterns))]

	// Noise sangat rendah agar pattern jelas terlihat
	noiseLevel := 0.15

	// Pattern selalu resolve di tutorial
	willResolve := true

	basePrice := 100.0

	allCandles := pattern.Build(basePrice, noiseLevel, willResolve, rng)

	observation := allCandles[:observationCandles]
	resolution := allCandles[observationCandles:]

	news := generateNewsEvents(pattern.ExpectedDirection(), rng)
	indicators := calculateIndicators(allCandles)
	observationIndicators := truncateIndicators(indicators, observationCandles)

	return &ChartSession{
		Seed:            seed,
		PatternType:     pattern.Type(),
		PatternName:     pattern.Name(),
		Difficulty:      "tutorial",
		ExpectedDir:     pattern.ExpectedDirection(),
		WillResolve:     true,
		NoiseLevel:      noiseLevel,
		ObservationData: observation,
		ResolutionData:  resolution,
		NewsEvents:      news,
		Indicators:      observationIndicators,
	}
}

// ─────────────────────────────────────────────────────────────
// SEED GENERATION
// ─────────────────────────────────────────────────────────────

// deriveSeed menghasilkan seed deterministik dari JID + timestamp.
// Menggunakan SHA-256 agar distribusi seed merata dan tidak predictable.
func deriveSeed(jid string) int64 {
	now := time.Now().UnixNano()
	data := []byte(jid)
	data = binary.BigEndian.AppendUint64(data, uint64(now))

	hash := sha256.Sum256(data)
	return int64(binary.BigEndian.Uint64(hash[:8]))
}

// ─────────────────────────────────────────────────────────────
// CANDLE HELPERS
// Fungsi-fungsi untuk membangun candlestick dari waypoints.
// ─────────────────────────────────────────────────────────────

// interpolateCandles menghasilkan candlestick antara waypoints.
//
// Setiap candle:
//   - Trend ke arah waypoint berikutnya
//   - Noise gaussian proporsional ke noiseLevel
//   - Mean-reversion mencegah harga menyimpang terlalu jauh dari jalur pattern
//   - Volume bervariasi (spike di key levels)
//
// waypoints: [(candle_index, target_price), ...]
// volumeSpikes: candle indices di mana volume harus tinggi
func interpolateCandles(
	startCandle int,
	count int,
	waypoints []waypoint,
	noiseLevel float64,
	baseVolume float64,
	volumeSpikes []int,
	rng *rand.Rand,
) []PricePoint {

	candles := make([]PricePoint, count)

	// Bangun expected price curve dari waypoints via linear interpolation
	expectedPrices := linearInterpolateWaypoints(waypoints, count)

	// Amplitude noise relatif terhadap range harga pattern
	priceRange := priceRangeOf(waypoints)
	noiseAmplitude := priceRange * noiseLevel * 0.04 // per-candle noise

	// Wick size proporsional
	wickSize := priceRange * 0.008

	currentPrice := expectedPrices[0]

	for i := 0; i < count; i++ {
		expected := expectedPrices[i]

		// Trend pull: tarik ke expected price (mean reversion)
		reversionStrength := 0.25
		trend := (expected - currentPrice) * reversionStrength

		// Random noise
		noise := rng.NormFloat64() * noiseAmplitude

		// Close price
		close := currentPrice + trend + noise
		if close <= 0 {
			close = currentPrice * 0.99 // jangan sampai negatif
		}

		open := currentPrice

		// High dan Low (wicks)
		high := math.Max(open, close) + math.Abs(rng.NormFloat64()*wickSize)
		low := math.Min(open, close) - math.Abs(rng.NormFloat64()*wickSize)
		if low <= 0 {
			low = math.Min(open, close) * 0.995
		}

		// Volume
		vol := baseVolume * (0.7 + rng.Float64()*0.6) // 70%-130% base
		for _, spike := range volumeSpikes {
			if i >= spike-1 && i <= spike+1 {
				vol *= 1.8 + rng.Float64()*1.2 // 180%-300% spike
			}
		}

		candles[i] = PricePoint{
			Time:   startCandle + i,
			Open:   roundPrice(open),
			High:   roundPrice(high),
			Low:    roundPrice(low),
			Close:  roundPrice(close),
			Volume: math.Round(vol),
		}

		currentPrice = close
	}

	return candles
}

// waypoint merepresentasikan titik target harga pada candle tertentu.
type waypoint struct {
	CandleIndex int
	Price       float64
}

// linearInterpolateWaypoints menghasilkan expected price di setiap candle
// via interpolasi linier antar waypoints.
func linearInterpolateWaypoints(wps []waypoint, totalCandles int) []float64 {
	prices := make([]float64, totalCandles)

	if len(wps) == 0 {
		return prices
	}

	// Isi sebelum waypoint pertama
	for i := 0; i < wps[0].CandleIndex && i < totalCandles; i++ {
		prices[i] = wps[0].Price
	}

	// Interpolasi antar waypoints
	for w := 0; w < len(wps)-1; w++ {
		from := wps[w]
		to := wps[w+1]
		span := to.CandleIndex - from.CandleIndex
		if span <= 0 {
			continue
		}

		for i := from.CandleIndex; i <= to.CandleIndex && i < totalCandles; i++ {
			t := float64(i-from.CandleIndex) / float64(span)
			prices[i] = from.Price + (to.Price-from.Price)*t
		}
	}

	// Isi setelah waypoint terakhir
	lastWP := wps[len(wps)-1]
	for i := lastWP.CandleIndex; i < totalCandles; i++ {
		prices[i] = lastWP.Price
	}

	return prices
}

// ─────────────────────────────────────────────────────────────
// INDICATOR CALCULATIONS
// ─────────────────────────────────────────────────────────────

// calculateIndicators menghitung MA20, RSI 14, dan MACD dari price data.
func calculateIndicators(candles []PricePoint) ChartIndicators {
	n := len(candles)
	closes := make([]float64, n)
	for i, c := range candles {
		closes[i] = c.Close
	}

	return ChartIndicators{
		MA20:    calcMA(closes, 20),
		RSI:     calcRSI(closes, 14),
		MACD:    calcMACD(closes),
		MACDSig: calcMACDSignal(calcMACD(closes)),
	}
}

// truncateIndicators memotong indikator agar hanya mencakup observation phase.
func truncateIndicators(ind ChartIndicators, count int) ChartIndicators {
	trunc := func(s []float64) []float64 {
		if len(s) > count {
			return s[:count]
		}
		return s
	}
	return ChartIndicators{
		MA20:    trunc(ind.MA20),
		RSI:     trunc(ind.RSI),
		MACD:    trunc(ind.MACD),
		MACDSig: trunc(ind.MACDSig),
	}
}

// calcMA menghitung Simple Moving Average.
func calcMA(data []float64, period int) []float64 {
	n := len(data)
	result := make([]float64, n)

	for i := range n {
		if i < period-1 {
			// Belum cukup data — gunakan rata-rata data yang ada
			sum := 0.0
			for j := 0; j <= i; j++ {
				sum += data[j]
			}
			result[i] = sum / float64(i+1)
		} else {
			sum := 0.0
			for j := i - period + 1; j <= i; j++ {
				sum += data[j]
			}
			result[i] = sum / float64(period)
		}
	}
	return result
}

// calcRSI menghitung Relative Strength Index.
func calcRSI(data []float64, period int) []float64 {
	n := len(data)
	result := make([]float64, n)

	if n < 2 {
		return result
	}

	// Hitung gains dan losses
	gains := make([]float64, n)
	losses := make([]float64, n)
	for i := 1; i < n; i++ {
		change := data[i] - data[i-1]
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = -change
		}
	}

	// Initial average (SMA of first `period` values)
	var avgGain, avgLoss float64
	for i := 1; i <= period && i < n; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	for i := 1; i < n; i++ {
		if i <= period {
			result[i] = 50 // default saat belum cukup data
			continue
		}

		// Smoothed average (EMA-style)
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)

		if avgLoss == 0 {
			result[i] = 100
		} else {
			rs := avgGain / avgLoss
			result[i] = 100 - (100 / (1 + rs))
		}
	}

	return result
}

// calcEMA menghitung Exponential Moving Average.
func calcEMA(data []float64, period int) []float64 {
	n := len(data)
	result := make([]float64, n)

	if n == 0 {
		return result
	}

	multiplier := 2.0 / float64(period+1)
	result[0] = data[0]

	for i := 1; i < n; i++ {
		result[i] = (data[i]-result[i-1])*multiplier + result[i-1]
	}

	return result
}

// calcMACD menghitung MACD line (EMA12 - EMA26).
func calcMACD(data []float64) []float64 {
	ema12 := calcEMA(data, 12)
	ema26 := calcEMA(data, 26)

	n := len(data)
	macd := make([]float64, n)
	for i := range n {
		macd[i] = ema12[i] - ema26[i]
	}
	return macd
}

// calcMACDSignal menghitung MACD Signal line (EMA9 of MACD).
func calcMACDSignal(macd []float64) []float64 {
	return calcEMA(macd, 9)
}

// ─────────────────────────────────────────────────────────────
// UTILITY HELPERS
// ─────────────────────────────────────────────────────────────

func roundPrice(p float64) float64 {
	return math.Round(p*100) / 100
}

func priceRangeOf(wps []waypoint) float64 {
	if len(wps) == 0 {
		return 1
	}
	min, max := wps[0].Price, wps[0].Price
	for _, wp := range wps {
		if wp.Price < min {
			min = wp.Price
		}
		if wp.Price > max {
			max = wp.Price
		}
	}
	r := max - min
	if r < 1 {
		return 1
	}
	return r
}
