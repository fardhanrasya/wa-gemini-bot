package trading

import "math/rand"

// Pattern mendefinisikan interface untuk semua pattern chart.
// Setiap pattern menghasilkan data harga berupa waypoints yang
// kemudian di-interpolasi menjadi candlestick oleh chart engine.
type Pattern interface {
	// Type mengembalikan identifier unik pattern (snake_case).
	Type() string
	// Name mengembalikan nama tampilan pattern.
	Name() string
	// ExpectedDirection mengembalikan arah yang diharapkan saat pattern resolve.
	ExpectedDirection() Direction
	// Difficulty mengembalikan tingkat kesulitan: "easy", "medium", "hard".
	Difficulty() string
	// Build menghasilkan seluruh candlestick data (observation + resolution).
	// willResolve menentukan apakah pattern resolve sesuai expected direction.
	Build(basePrice, noiseLevel float64, willResolve bool, rng *rand.Rand) []PricePoint
}

// allPatterns berisi semua pattern yang tersedia di game.
// Digunakan oleh selectPattern untuk memilih secara random.
var allPatterns = []Pattern{
	// Easy (4)
	&DoubleBottomPattern{},
	&DoubleTopPattern{},
	&BullFlagPattern{},
	&BearFlagPattern{},
	// Medium (11)
	&HeadAndShouldersPattern{},
	&InverseHnSPattern{},
	&AscendingTrianglePattern{},
	&DescendingTrianglePattern{},
	&TripleBottomPattern{},
	&TripleTopPattern{},
	&BullishPennantPattern{},
	&BearishPennantPattern{},
	&ChannelUpPattern{},
	&ChannelDownPattern{},
	&VRecoveryPattern{},
	// Hard (8)
	&SpikeTopPattern{},
	&CupAndHandlePattern{},
	&RisingWedgePattern{},
	&FallingWedgePattern{},
	&RoundingBottomPattern{},
	&SymmetricalTrianglePattern{},
	&FakeoutRallyPattern{},
	&FakeoutDropPattern{},
}

// selectPattern memilih pattern secara random dengan weighted distribution.
// Easy patterns muncul lebih sering agar game tetap accessible,
// tapi hard patterns tetap muncul cukup sering agar menantang.
func selectPattern(rng *rand.Rand) Pattern {
	// Weighted: easy=35%, medium=40%, hard=25%
	roll := rng.Float64()
	var pool []Pattern
	switch {
	case roll < 0.35:
		pool = filterByDifficulty("easy")
	case roll < 0.75:
		pool = filterByDifficulty("medium")
	default:
		pool = filterByDifficulty("hard")
	}
	if len(pool) == 0 {
		pool = allPatterns
	}
	return pool[rng.Intn(len(pool))]
}

func filterByDifficulty(diff string) []Pattern {
	var result []Pattern
	for _, p := range allPatterns {
		if p.Difficulty() == diff {
			result = append(result, p)
		}
	}
	return result
}

// ─────────────────────────────────────────────────────────────
// HELPER: buildFromWaypoints
// Shorthand yang digunakan oleh semua pattern untuk menghasilkan
// candlestick dari daftar waypoints + volume spike locations.
// ─────────────────────────────────────────────────────────────

func buildFromWaypoints(wps []waypoint, noiseLevel float64, volumeSpikes []int, rng *rand.Rand) []PricePoint {
	baseVolume := 500.0 + rng.Float64()*500.0
	return interpolateCandles(0, totalCandles, wps, noiseLevel, baseVolume, volumeSpikes, rng)
}

// resolveDir mengembalikan direction yang akan terjadi di resolution,
// mempertimbangkan apakah pattern resolve benar atau gagal.
func resolveDir(expected Direction, willResolve bool) Direction {
	if willResolve {
		return expected
	}
	if expected == DirBullish {
		return DirBearish
	}
	return DirBullish
}

// ═══════════════════════════════════════════════════════════════
// EASY PATTERNS
// ═══════════════════════════════════════════════════════════════

// ─── 1. Double Bottom (Bullish) ───

type DoubleBottomPattern struct{}

func (p *DoubleBottomPattern) Type() string              { return "double_bottom" }
func (p *DoubleBottomPattern) Name() string              { return "Double Bottom" }
func (p *DoubleBottomPattern) ExpectedDirection() Direction { return DirBullish }
func (p *DoubleBottomPattern) Difficulty() string         { return "easy" }

func (p *DoubleBottomPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	drop := base * (0.08 + rng.Float64()*0.06) // 8-14% decline
	support := base - drop
	bounceHeight := drop * (0.4 + rng.Float64()*0.2) // bounce 40-60% of drop

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = base + drop*(0.3+rng.Float64()*0.4) // breakout above base
	} else {
		resolvePrice = support - drop*(0.3+rng.Float64()*0.3) // breakdown below support
	}

	wps := []waypoint{
		{0, base},
		{10, support},                  // first bottom
		{20, support + bounceHeight},   // bounce
		{30, support + support*0.005},  // second bottom (roughly same level)
		{38, support + bounceHeight*0.6}, // start of recovery
		{observationCandles, support + bounceHeight*0.8},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{10, 30, observationCandles + 5}, rng)
}

// ─── 2. Double Top (Bearish) ───

type DoubleTopPattern struct{}

func (p *DoubleTopPattern) Type() string              { return "double_top" }
func (p *DoubleTopPattern) Name() string              { return "Double Top" }
func (p *DoubleTopPattern) ExpectedDirection() Direction { return DirBearish }
func (p *DoubleTopPattern) Difficulty() string         { return "easy" }

func (p *DoubleTopPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	rise := base * (0.08 + rng.Float64()*0.06)
	resistance := base + rise
	dip := rise * (0.4 + rng.Float64()*0.2)

	dir := resolveDir(DirBearish, resolve)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = base - rise*(0.3+rng.Float64()*0.4)
	} else {
		resolvePrice = resistance + rise*(0.3+rng.Float64()*0.3)
	}

	wps := []waypoint{
		{0, base},
		{10, resistance},
		{20, resistance - dip},
		{30, resistance - resistance*0.003},
		{38, resistance - dip*0.5},
		{observationCandles, resistance - dip*0.7},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{10, 30, observationCandles + 5}, rng)
}

// ─── 3. Bull Flag (Bullish) ───

type BullFlagPattern struct{}

func (p *BullFlagPattern) Type() string              { return "bull_flag" }
func (p *BullFlagPattern) Name() string              { return "Bull Flag" }
func (p *BullFlagPattern) ExpectedDirection() Direction { return DirBullish }
func (p *BullFlagPattern) Difficulty() string         { return "easy" }

func (p *BullFlagPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	rallySize := base * (0.12 + rng.Float64()*0.08)
	top := base + rallySize
	pullback := rallySize * (0.3 + rng.Float64()*0.2)
	flagBottom := top - pullback

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = top + rallySize*(0.4+rng.Float64()*0.3)
	} else {
		resolvePrice = flagBottom - rallySize*(0.5+rng.Float64()*0.3)
	}

	wps := []waypoint{
		{0, base},
		{12, top},                      // sharp rally (flagpole)
		{18, top - pullback*0.3},       // slight pullback
		{25, top - pullback*0.6},       // continuing pullback (flag forming)
		{32, flagBottom},               // flag bottom
		{38, flagBottom + pullback*0.2}, // slight uptick at end of flag
		{observationCandles, flagBottom + pullback*0.3},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{6, 12, observationCandles + 3}, rng)
}

// ─── 4. Bear Flag (Bearish) ───

type BearFlagPattern struct{}

func (p *BearFlagPattern) Type() string              { return "bear_flag" }
func (p *BearFlagPattern) Name() string              { return "Bear Flag" }
func (p *BearFlagPattern) ExpectedDirection() Direction { return DirBearish }
func (p *BearFlagPattern) Difficulty() string         { return "easy" }

func (p *BearFlagPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	dropSize := base * (0.12 + rng.Float64()*0.08)
	bottom := base - dropSize
	bounce := dropSize * (0.3 + rng.Float64()*0.2)
	flagTop := bottom + bounce

	dir := resolveDir(DirBearish, resolve)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = bottom - dropSize*(0.4+rng.Float64()*0.3)
	} else {
		resolvePrice = flagTop + dropSize*(0.5+rng.Float64()*0.3)
	}

	wps := []waypoint{
		{0, base},
		{12, bottom},
		{18, bottom + bounce*0.3},
		{25, bottom + bounce*0.6},
		{32, flagTop},
		{38, flagTop - bounce*0.2},
		{observationCandles, flagTop - bounce*0.3},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{6, 12, observationCandles + 3}, rng)
}

// ═══════════════════════════════════════════════════════════════
// MEDIUM PATTERNS
// ═══════════════════════════════════════════════════════════════

// ─── 5. Head & Shoulders (Bearish) ───

type HeadAndShouldersPattern struct{}

func (p *HeadAndShouldersPattern) Type() string              { return "head_and_shoulders" }
func (p *HeadAndShouldersPattern) Name() string              { return "Head & Shoulders" }
func (p *HeadAndShouldersPattern) ExpectedDirection() Direction { return DirBearish }
func (p *HeadAndShouldersPattern) Difficulty() string         { return "medium" }

func (p *HeadAndShouldersPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	shoulderH := base * (0.06 + rng.Float64()*0.04)
	headH := shoulderH * (1.4 + rng.Float64()*0.3)
	neckline := base - base*0.02

	dir := resolveDir(DirBearish, resolve)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = neckline - headH*(0.5+rng.Float64()*0.3)
	} else {
		resolvePrice = base + headH*(0.6+rng.Float64()*0.3)
	}

	wps := []waypoint{
		{0, base},
		{6, base + shoulderH},   // left shoulder peak
		{11, neckline},          // neckline
		{18, base + headH},      // head peak
		{25, neckline},          // neckline
		{32, base + shoulderH*0.9}, // right shoulder (slightly lower)
		{38, neckline + neckline*0.01},
		{observationCandles, neckline},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{18, 32, observationCandles + 3}, rng)
}

// ─── 6. Inverse Head & Shoulders (Bullish) ───

type InverseHnSPattern struct{}

func (p *InverseHnSPattern) Type() string              { return "inverse_hns" }
func (p *InverseHnSPattern) Name() string              { return "Inverse Head & Shoulders" }
func (p *InverseHnSPattern) ExpectedDirection() Direction { return DirBullish }
func (p *InverseHnSPattern) Difficulty() string         { return "medium" }

func (p *InverseHnSPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	shoulderD := base * (0.06 + rng.Float64()*0.04)
	headD := shoulderD * (1.4 + rng.Float64()*0.3)
	neckline := base + base*0.02

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = neckline + headD*(0.5+rng.Float64()*0.3)
	} else {
		resolvePrice = base - headD*(0.6+rng.Float64()*0.3)
	}

	wps := []waypoint{
		{0, base},
		{6, base - shoulderD},
		{11, neckline},
		{18, base - headD},
		{25, neckline},
		{32, base - shoulderD*0.9},
		{38, neckline - neckline*0.005},
		{observationCandles, neckline},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{18, 32, observationCandles + 3}, rng)
}

// ─── 7. Ascending Triangle (Bullish) ───

type AscendingTrianglePattern struct{}

func (p *AscendingTrianglePattern) Type() string              { return "ascending_triangle" }
func (p *AscendingTrianglePattern) Name() string              { return "Ascending Triangle" }
func (p *AscendingTrianglePattern) ExpectedDirection() Direction { return DirBullish }
func (p *AscendingTrianglePattern) Difficulty() string         { return "medium" }

func (p *AscendingTrianglePattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	resistance := base + base*(0.05+rng.Float64()*0.03)
	lowStart := base - base*(0.04+rng.Float64()*0.02)

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = resistance + (resistance-lowStart)*(0.4+rng.Float64()*0.3)
	} else {
		resolvePrice = lowStart - (resistance-lowStart)*(0.3+rng.Float64()*0.2)
	}

	// Higher lows converging toward flat resistance
	wps := []waypoint{
		{0, base},
		{5, resistance},
		{10, lowStart},
		{16, resistance - resistance*0.003},
		{22, lowStart + (resistance-lowStart)*0.3},
		{28, resistance - resistance*0.005},
		{34, lowStart + (resistance-lowStart)*0.55},
		{38, resistance - resistance*0.008},
		{observationCandles, resistance - resistance*0.01},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{5, 16, 28, observationCandles + 3}, rng)
}

// ─── 8. Descending Triangle (Bearish) ───

type DescendingTrianglePattern struct{}

func (p *DescendingTrianglePattern) Type() string              { return "descending_triangle" }
func (p *DescendingTrianglePattern) Name() string              { return "Descending Triangle" }
func (p *DescendingTrianglePattern) ExpectedDirection() Direction { return DirBearish }
func (p *DescendingTrianglePattern) Difficulty() string         { return "medium" }

func (p *DescendingTrianglePattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	support := base - base*(0.05+rng.Float64()*0.03)
	highStart := base + base*(0.04+rng.Float64()*0.02)

	dir := resolveDir(DirBearish, resolve)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = support - (highStart-support)*(0.4+rng.Float64()*0.3)
	} else {
		resolvePrice = highStart + (highStart-support)*(0.3+rng.Float64()*0.2)
	}

	wps := []waypoint{
		{0, base},
		{5, support},
		{10, highStart},
		{16, support + support*0.003},
		{22, highStart - (highStart-support)*0.3},
		{28, support + support*0.005},
		{34, highStart - (highStart-support)*0.55},
		{38, support + support*0.008},
		{observationCandles, support + support*0.01},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{5, 16, 28, observationCandles + 3}, rng)
}

// ─── 9. Triple Bottom (Bullish) ───

type TripleBottomPattern struct{}

func (p *TripleBottomPattern) Type() string              { return "triple_bottom" }
func (p *TripleBottomPattern) Name() string              { return "Triple Bottom" }
func (p *TripleBottomPattern) ExpectedDirection() Direction { return DirBullish }
func (p *TripleBottomPattern) Difficulty() string         { return "medium" }

func (p *TripleBottomPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	drop := base * (0.08 + rng.Float64()*0.05)
	support := base - drop
	bounceH := drop * (0.5 + rng.Float64()*0.2)

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = base + drop*(0.3+rng.Float64()*0.3)
	} else {
		resolvePrice = support - drop*(0.3+rng.Float64()*0.2)
	}

	wps := []waypoint{
		{0, base},
		{8, support},                 // 1st bottom
		{14, support + bounceH},
		{20, support + support*0.005}, // 2nd bottom
		{26, support + bounceH*0.8},
		{33, support + support*0.003}, // 3rd bottom
		{38, support + bounceH*0.5},
		{observationCandles, support + bounceH*0.7},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{8, 20, 33, observationCandles + 3}, rng)
}

// ─── 10. Triple Top (Bearish) ───

type TripleTopPattern struct{}

func (p *TripleTopPattern) Type() string              { return "triple_top" }
func (p *TripleTopPattern) Name() string              { return "Triple Top" }
func (p *TripleTopPattern) ExpectedDirection() Direction { return DirBearish }
func (p *TripleTopPattern) Difficulty() string         { return "medium" }

func (p *TripleTopPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	rise := base * (0.08 + rng.Float64()*0.05)
	resistance := base + rise
	dipH := rise * (0.5 + rng.Float64()*0.2)

	dir := resolveDir(DirBearish, resolve)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = base - rise*(0.3+rng.Float64()*0.3)
	} else {
		resolvePrice = resistance + rise*(0.3+rng.Float64()*0.2)
	}

	wps := []waypoint{
		{0, base},
		{8, resistance},
		{14, resistance - dipH},
		{20, resistance - resistance*0.003},
		{26, resistance - dipH*0.8},
		{33, resistance - resistance*0.005},
		{38, resistance - dipH*0.5},
		{observationCandles, resistance - dipH*0.7},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{8, 20, 33, observationCandles + 3}, rng)
}

// ─── 11. Bullish Pennant ───

type BullishPennantPattern struct{}

func (p *BullishPennantPattern) Type() string              { return "bullish_pennant" }
func (p *BullishPennantPattern) Name() string              { return "Bullish Pennant" }
func (p *BullishPennantPattern) ExpectedDirection() Direction { return DirBullish }
func (p *BullishPennantPattern) Difficulty() string         { return "medium" }

func (p *BullishPennantPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	rally := base * (0.10 + rng.Float64()*0.06)
	top := base + rally
	convergence := rally * 0.4

	dir := resolveDir(DirBullish, resolve)
	midPennant := top - convergence*0.5
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = top + rally*(0.4+rng.Float64()*0.3)
	} else {
		resolvePrice = midPennant - rally*(0.5+rng.Float64()*0.3)
	}

	wps := []waypoint{
		{0, base},
		{10, top},                               // flagpole
		{16, top - convergence*0.5},              // pennant lower
		{22, top - convergence*0.15},             // pennant upper
		{28, top - convergence*0.45},             // converging lower
		{34, top - convergence*0.3},              // converging upper
		{38, midPennant},                         // apex
		{observationCandles, midPennant + midPennant*0.005},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{5, 10, observationCandles + 3}, rng)
}

// ─── 12. Bearish Pennant ───

type BearishPennantPattern struct{}

func (p *BearishPennantPattern) Type() string              { return "bearish_pennant" }
func (p *BearishPennantPattern) Name() string              { return "Bearish Pennant" }
func (p *BearishPennantPattern) ExpectedDirection() Direction { return DirBearish }
func (p *BearishPennantPattern) Difficulty() string         { return "medium" }

func (p *BearishPennantPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	drop := base * (0.10 + rng.Float64()*0.06)
	bottom := base - drop
	convergence := drop * 0.4

	dir := resolveDir(DirBearish, resolve)
	midPennant := bottom + convergence*0.5
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = bottom - drop*(0.4+rng.Float64()*0.3)
	} else {
		resolvePrice = midPennant + drop*(0.5+rng.Float64()*0.3)
	}

	wps := []waypoint{
		{0, base},
		{10, bottom},
		{16, bottom + convergence*0.5},
		{22, bottom + convergence*0.15},
		{28, bottom + convergence*0.45},
		{34, bottom + convergence*0.3},
		{38, midPennant},
		{observationCandles, midPennant - midPennant*0.003},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{5, 10, observationCandles + 3}, rng)
}

// ─── 13. Channel Up (Bullish) ───

type ChannelUpPattern struct{}

func (p *ChannelUpPattern) Type() string              { return "channel_up" }
func (p *ChannelUpPattern) Name() string              { return "Channel Up" }
func (p *ChannelUpPattern) ExpectedDirection() Direction { return DirBullish }
func (p *ChannelUpPattern) Difficulty() string         { return "medium" }

func (p *ChannelUpPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	slope := base * (0.002 + rng.Float64()*0.002) // per-candle upward slope
	channelWidth := base * (0.04 + rng.Float64()*0.03)

	dir := resolveDir(DirBullish, resolve)
	endBase := base + slope*float64(totalCandles)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = endBase + channelWidth*(0.5+rng.Float64()*0.5) // breakout above
	} else {
		resolvePrice = endBase - channelWidth*(1.0+rng.Float64()*0.5) // breakdown below
	}

	// Zigzag within channel
	wps := []waypoint{
		{0, base},
		{5, base + slope*5 + channelWidth*0.5},  // top of channel
		{12, base + slope*12 - channelWidth*0.3}, // bottom bounce
		{18, base + slope*18 + channelWidth*0.4},
		{25, base + slope*25 - channelWidth*0.2},
		{32, base + slope*32 + channelWidth*0.3},
		{38, base + slope*38 - channelWidth*0.1},
		{observationCandles, base + slope*float64(observationCandles)},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{12, 25, observationCandles + 5}, rng)
}

// ─── 14. Channel Down (Bearish) ───

type ChannelDownPattern struct{}

func (p *ChannelDownPattern) Type() string              { return "channel_down" }
func (p *ChannelDownPattern) Name() string              { return "Channel Down" }
func (p *ChannelDownPattern) ExpectedDirection() Direction { return DirBearish }
func (p *ChannelDownPattern) Difficulty() string         { return "medium" }

func (p *ChannelDownPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	slope := base * (0.002 + rng.Float64()*0.002)
	channelWidth := base * (0.04 + rng.Float64()*0.03)

	dir := resolveDir(DirBearish, resolve)
	endBase := base - slope*float64(totalCandles)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = endBase - channelWidth*(0.5+rng.Float64()*0.5)
	} else {
		resolvePrice = endBase + channelWidth*(1.0+rng.Float64()*0.5)
	}

	wps := []waypoint{
		{0, base},
		{5, base - slope*5 - channelWidth*0.5},
		{12, base - slope*12 + channelWidth*0.3},
		{18, base - slope*18 - channelWidth*0.4},
		{25, base - slope*25 + channelWidth*0.2},
		{32, base - slope*32 - channelWidth*0.3},
		{38, base - slope*38 + channelWidth*0.1},
		{observationCandles, base - slope*float64(observationCandles)},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{12, 25, observationCandles + 5}, rng)
}

// ─── 15. V-Recovery (Bullish) ───

type VRecoveryPattern struct{}

func (p *VRecoveryPattern) Type() string              { return "v_recovery" }
func (p *VRecoveryPattern) Name() string              { return "V-Recovery" }
func (p *VRecoveryPattern) ExpectedDirection() Direction { return DirBullish }
func (p *VRecoveryPattern) Difficulty() string         { return "medium" }

func (p *VRecoveryPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	crash := base * (0.15 + rng.Float64()*0.10)
	bottom := base - crash

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = base + crash*(0.2+rng.Float64()*0.3)
	} else {
		resolvePrice = bottom + crash*0.1 // fake recovery then drop
	}

	wps := []waypoint{
		{0, base},
		{15, bottom},                // sharp V bottom
		{30, base - crash*0.2},      // recovered most of it
		{38, base - crash*0.1},
		{observationCandles, base - crash*0.05},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{14, 15, 16, observationCandles + 3}, rng)
}

// ═══════════════════════════════════════════════════════════════
// HARD PATTERNS
// ═══════════════════════════════════════════════════════════════

// ─── 16. Spike Top (Bearish) ───

type SpikeTopPattern struct{}

func (p *SpikeTopPattern) Type() string              { return "spike_top" }
func (p *SpikeTopPattern) Name() string              { return "Spike Top" }
func (p *SpikeTopPattern) ExpectedDirection() Direction { return DirBearish }
func (p *SpikeTopPattern) Difficulty() string         { return "hard" }

func (p *SpikeTopPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	spike := base * (0.15 + rng.Float64()*0.10)
	top := base + spike

	dir := resolveDir(DirBearish, resolve)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = base - spike*(0.3+rng.Float64()*0.3)
	} else {
		resolvePrice = top - spike*0.1
	}

	wps := []waypoint{
		{0, base},
		{15, top},
		{30, base + spike*0.2},
		{38, base + spike*0.15},
		{observationCandles, base + spike*0.1},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{14, 15, 16, observationCandles + 3}, rng)
}

// ─── 17. Cup & Handle (Bullish) ───

type CupAndHandlePattern struct{}

func (p *CupAndHandlePattern) Type() string              { return "cup_and_handle" }
func (p *CupAndHandlePattern) Name() string              { return "Cup & Handle" }
func (p *CupAndHandlePattern) ExpectedDirection() Direction { return DirBullish }
func (p *CupAndHandlePattern) Difficulty() string         { return "hard" }

func (p *CupAndHandlePattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	cupDepth := base * (0.08 + rng.Float64()*0.06)
	cupBottom := base - cupDepth
	handleDip := cupDepth * (0.3 + rng.Float64()*0.15)

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = base + cupDepth*(0.5+rng.Float64()*0.3)
	} else {
		resolvePrice = base - cupDepth*(0.4+rng.Float64()*0.3)
	}

	wps := []waypoint{
		{0, base},
		{5, base - cupDepth*0.5},  // cup left side
		{14, cupBottom},           // cup bottom (gradual U)
		{22, base - cupDepth*0.4}, // cup right side rising
		{28, base - base*0.01},    // back near rim
		{33, base - handleDip},    // handle dip
		{38, base - handleDip*0.5},
		{observationCandles, base - handleDip*0.3},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{14, 28, 33, observationCandles + 3}, rng)
}

// ─── 18. Rising Wedge (Bearish) ───

type RisingWedgePattern struct{}

func (p *RisingWedgePattern) Type() string              { return "rising_wedge" }
func (p *RisingWedgePattern) Name() string              { return "Rising Wedge" }
func (p *RisingWedgePattern) ExpectedDirection() Direction { return DirBearish }
func (p *RisingWedgePattern) Difficulty() string         { return "hard" }

func (p *RisingWedgePattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	upperSlope := base * 0.003
	lowerSlope := base * 0.0045 // lower line rising faster = converging

	dir := resolveDir(DirBearish, resolve)
	apex := base + upperSlope*float64(observationCandles)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = apex - base*(0.10+rng.Float64()*0.06)
	} else {
		resolvePrice = apex + base*(0.05+rng.Float64()*0.04)
	}

	width := base * 0.04
	wps := []waypoint{
		{0, base},
		{6, base + upperSlope*6},
		{12, base + lowerSlope*12 - width*0.3},
		{18, base + upperSlope*18 + width*0.1},
		{24, base + lowerSlope*24 - width*0.15},
		{30, base + upperSlope*30 + width*0.05},
		{36, base + lowerSlope*36 - width*0.05},
		{observationCandles, apex},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{30, 36, observationCandles + 3}, rng)
}

// ─── 19. Falling Wedge (Bullish) ───

type FallingWedgePattern struct{}

func (p *FallingWedgePattern) Type() string              { return "falling_wedge" }
func (p *FallingWedgePattern) Name() string              { return "Falling Wedge" }
func (p *FallingWedgePattern) ExpectedDirection() Direction { return DirBullish }
func (p *FallingWedgePattern) Difficulty() string         { return "hard" }

func (p *FallingWedgePattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	upperSlope := base * 0.0045
	lowerSlope := base * 0.003

	dir := resolveDir(DirBullish, resolve)
	apex := base - lowerSlope*float64(observationCandles)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = apex + base*(0.10+rng.Float64()*0.06)
	} else {
		resolvePrice = apex - base*(0.05+rng.Float64()*0.04)
	}

	width := base * 0.04
	wps := []waypoint{
		{0, base},
		{6, base - lowerSlope*6},
		{12, base - upperSlope*12 + width*0.3},
		{18, base - lowerSlope*18 - width*0.1},
		{24, base - upperSlope*24 + width*0.15},
		{30, base - lowerSlope*30 - width*0.05},
		{36, base - upperSlope*36 + width*0.05},
		{observationCandles, apex},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{30, 36, observationCandles + 3}, rng)
}

// ─── 20. Rounding Bottom (Bullish) ───

type RoundingBottomPattern struct{}

func (p *RoundingBottomPattern) Type() string              { return "rounding_bottom" }
func (p *RoundingBottomPattern) Name() string              { return "Rounding Bottom" }
func (p *RoundingBottomPattern) ExpectedDirection() Direction { return DirBullish }
func (p *RoundingBottomPattern) Difficulty() string         { return "hard" }

func (p *RoundingBottomPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	depth := base * (0.08 + rng.Float64()*0.05)

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = base + depth*(0.4+rng.Float64()*0.3)
	} else {
		resolvePrice = base - depth*(0.5+rng.Float64()*0.3)
	}

	// Gradual U-curve using quadratic interpolation
	// price = base - depth * (1 - ((t - center) / halfSpan)^2)
	wps := make([]waypoint, 0, 10)
	center := float64(observationCandles) * 0.5
	halfSpan := center
	for i := 0; i <= observationCandles; i += 5 {
		t := float64(i)
		normalized := (t - center) / halfSpan
		price := base - depth*(1-normalized*normalized)
		wps = append(wps, waypoint{i, price})
	}
	wps = append(wps, waypoint{totalCandles, resolvePrice})

	return buildFromWaypoints(wps, noise, []int{int(center), observationCandles + 3}, rng)
}

// ─── 21. Symmetrical Triangle (Ambiguous) ───

type SymmetricalTrianglePattern struct{}

func (p *SymmetricalTrianglePattern) Type() string              { return "symmetrical_triangle" }
func (p *SymmetricalTrianglePattern) Name() string              { return "Symmetrical Triangle" }
func (p *SymmetricalTrianglePattern) ExpectedDirection() Direction { return DirBullish } // slight bullish bias
func (p *SymmetricalTrianglePattern) Difficulty() string         { return "hard" }

func (p *SymmetricalTrianglePattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	amplitude := base * (0.06 + rng.Float64()*0.04)

	// For symmetrical triangle, direction is more random (50/50 with slight bullish bias)
	dir := resolveDir(DirBullish, resolve)
	// Override: 50/50 actual direction regardless of resolve flag
	if rng.Float64() < 0.5 {
		dir = DirBearish
	}

	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = base + amplitude*(0.5+rng.Float64()*0.4)
	} else {
		resolvePrice = base - amplitude*(0.5+rng.Float64()*0.4)
	}

	// Converging from both sides
	wps := []waypoint{
		{0, base},
		{5, base + amplitude*0.8},
		{10, base - amplitude*0.7},
		{16, base + amplitude*0.55},
		{22, base - amplitude*0.45},
		{28, base + amplitude*0.3},
		{34, base - amplitude*0.2},
		{38, base + amplitude*0.08},
		{observationCandles, base},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{observationCandles - 2, observationCandles + 3}, rng)
}

// ─── 22. Fakeout Rally (Bearish Trap) ───

type FakeoutRallyPattern struct{}

func (p *FakeoutRallyPattern) Type() string              { return "fakeout_rally" }
func (p *FakeoutRallyPattern) Name() string              { return "Fakeout Rally" }
func (p *FakeoutRallyPattern) ExpectedDirection() Direction { return DirBearish }
func (p *FakeoutRallyPattern) Difficulty() string         { return "hard" }

func (p *FakeoutRallyPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	fakeRally := base * (0.08 + rng.Float64()*0.06)
	fakeTop := base + fakeRally

	dir := resolveDir(DirBearish, resolve)
	var resolvePrice float64
	if dir == DirBearish {
		resolvePrice = base - fakeRally*(0.8+rng.Float64()*0.4) // reversal below base
	} else {
		resolvePrice = fakeTop + fakeRally*(0.2+rng.Float64()*0.2) // actual breakout
	}

	// Looks bullish during observation, but volume is thin (clue for skilled players)
	wps := []waypoint{
		{0, base},
		{8, base + fakeRally*0.3},
		{15, base + fakeRally*0.2},  // small pullback
		{22, base + fakeRally*0.6},  // looks like it's breaking out
		{30, fakeTop - fakeTop*0.005},
		{35, fakeTop},               // fake breakout!
		{38, fakeTop - fakeRally*0.05},
		{observationCandles, fakeTop - fakeRally*0.08},
		{48, base + fakeRally*0.2},  // starts reversing
		{totalCandles, resolvePrice},
	}

	// Low volume during "breakout" = clue that it's fake
	return buildFromWaypoints(wps, noise, []int{8, 22, 48}, rng)
}

// ─── 23. Fakeout Drop (Bullish Trap) ───

type FakeoutDropPattern struct{}

func (p *FakeoutDropPattern) Type() string              { return "fakeout_drop" }
func (p *FakeoutDropPattern) Name() string              { return "Fakeout Drop" }
func (p *FakeoutDropPattern) ExpectedDirection() Direction { return DirBullish }
func (p *FakeoutDropPattern) Difficulty() string         { return "hard" }

func (p *FakeoutDropPattern) Build(base, noise float64, resolve bool, rng *rand.Rand) []PricePoint {
	fakeDrop := base * (0.08 + rng.Float64()*0.06)
	fakeBottom := base - fakeDrop

	dir := resolveDir(DirBullish, resolve)
	var resolvePrice float64
	if dir == DirBullish {
		resolvePrice = base + fakeDrop*(0.8+rng.Float64()*0.4)
	} else {
		resolvePrice = fakeBottom - fakeDrop*(0.2+rng.Float64()*0.2)
	}

	wps := []waypoint{
		{0, base},
		{8, base - fakeDrop*0.3},
		{15, base - fakeDrop*0.2},
		{22, base - fakeDrop*0.6},
		{30, fakeBottom + fakeBottom*0.005},
		{35, fakeBottom},
		{38, fakeBottom + fakeDrop*0.05},
		{observationCandles, fakeBottom + fakeDrop*0.08},
		{48, base - fakeDrop*0.2},
		{totalCandles, resolvePrice},
	}

	return buildFromWaypoints(wps, noise, []int{8, 22, 48}, rng)
}
