package planner

import (
	"fmt"
	"math"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

// AdaptiveColor256 spreads up to 256 colors over the 16 hardware palettes
// and assigns every scanline to one of them through its SCB. "256" is the
// ceiling (16 palettes × 16 colors), not a promise — palettes repeat colors
// wherever the image demands it.
//
// The enemy of multi-palette output is banding: adjacent scanlines quantized
// through mismatched palettes render the same source color differently, and
// a horizontal seam is among the most perceptible artifacts there is. Each
// --scb-mode strategy trades palette freedom against seam risk differently:
//
//	single    one bestPalette16 palette for the whole frame (no seams, least
//	          colors — identical to shr320-color16 output)
//	banded    16 equal fixed bands of 12–13 lines, one palette each
//	grouped   16 contiguous bands with adaptive cut placement (the default):
//	          dynamic programming puts the 15 palette switches where the
//	          image content already changes, i.e. where a switch is least
//	          visible
//	per-line  fully dynamic: scanlines are clustered by content and any line
//	          may use any palette — the closest per-line match, relying on
//	          error diffusion to soften whatever seams that creates
//
// In the banded modes each palette also trains on a few rows past its band
// edges at reduced weight, so neighboring palettes agree about the colors
// that straddle their boundary — the cheapest banding countermeasure of all.
type AdaptiveColor256 struct{}

// NewAdaptiveColor256 returns the multi-palette color planner.
func NewAdaptiveColor256() *AdaptiveColor256 { return &AdaptiveColor256{} }

func (*AdaptiveColor256) Name() string { return "adaptive-color256" }

func (a *AdaptiveColor256) Plan(src *pix.LinearImage, opt pipeline.Options) (pipeline.Plan, error) {
	// Line assignments and band math assume the full prepared frame.
	if src.W != shr.Width320 || src.H != shr.Height {
		return pipeline.Plan{}, fmt.Errorf("planner input is %dx%d, want %dx%d",
			src.W, src.H, shr.Width320, shr.Height)
	}
	switch opt.SCBMode {
	case "single":
		var p pipeline.Plan
		p.Palettes[0] = bestPalette16(src)
		return p, nil
	case "banded":
		return bandedPlan(lineBinsOf(src), fixedCuts(src.H)), nil
	case "", "auto", "grouped":
		lb := lineBinsOf(src)
		return bandedPlan(lb, adaptiveCuts(lb)), nil
	case "per-line":
		return perLinePlan(lineBinsOf(src)), nil
	default:
		return pipeline.Plan{}, fmt.Errorf("--scb-mode %q not supported by planner %s", opt.SCBMode, a.Name())
	}
}

// lineBin aggregates one scanline's pixels sharing an RGB444 cell. Channel
// sums stay raw (not means) so bins from different lines merge by addition;
// the Oklab of the cell mean is cached for palette-error evaluation. As with
// cbin, a bin stands in for its pixels at their mean color — within-cell
// spread is below what the 4-bit hardware palette can express anyway.
type lineBin struct {
	key     int     // RGB444 cell, for merging across lines
	r, g, b float64 // sums of linear RGB over the member pixels
	n       float64 // pixel count
	L, A, B float64 // Oklab of the mean color
}

// lineBinsOf bins every scanline separately, cells in ascending key order.
func lineBinsOf(src *pix.LinearImage) [][]lineBin {
	out := make([][]lineBin, src.H)
	for y := 0; y < src.H; y++ {
		var cells [4096]lineBin
		for x := 0; x < src.W; x++ {
			i := 3 * (y*src.W + x)
			r := float64(src.Pix[i])
			g := float64(src.Pix[i+1])
			b := float64(src.Pix[i+2])
			c := &cells[int(nib(r))<<8|int(nib(g))<<4|int(nib(b))]
			c.r += r
			c.g += g
			c.b += b
			c.n++
		}
		bins := make([]lineBin, 0, 64)
		for key := range cells {
			c := cells[key]
			if c.n == 0 {
				continue
			}
			c.key = key
			c.L, c.A, c.B = pix.LinearToOklab(c.r/c.n, c.g/c.n, c.b/c.n)
			bins = append(bins, c)
		}
		out[y] = bins
	}
	return out
}

// binsFor merges per-line bins into whole-cluster weighted bins, scaling
// each line's contribution by rowWeight[y] (0 skips the line). The result
// feeds paletteFromBins exactly like a whole-image histogram would.
func binsFor(lb [][]lineBin, rowWeight []float64) []cbin {
	var cells [4096]cbin
	for y, bins := range lb {
		w := rowWeight[y]
		if w == 0 {
			continue
		}
		for i := range bins {
			b := &bins[i]
			c := &cells[b.key]
			c.r += w * b.r
			c.g += w * b.g
			c.b += w * b.b
			c.n += w * b.n
		}
	}
	out := make([]cbin, 0, 256)
	for key := range cells {
		c := cells[key]
		if c.n == 0 {
			continue
		}
		c.r /= c.n
		c.g /= c.n
		c.b /= c.n
		c.L, c.A, c.B = pix.LinearToOklab(c.r, c.g, c.b)
		out = append(out, c)
	}
	return out
}

// fixedCuts splits h rows into 16 near-equal contiguous bands; cuts[i] is
// the first row of band i, cuts[16] == h.
func fixedCuts(h int) []int {
	cuts := make([]int, 17)
	for i := range cuts {
		cuts[i] = i * h / 16
	}
	return cuts
}

// overlapRows is how far past each band edge palette training looks. The
// neighbor rows get less weight than the band's own (0.5 tapering to ~0.08)
// so adjacent palettes agree about boundary-straddling colors without the
// neighbors' content diluting the band's own.
const overlapRows = 6

func bandWeights(h, a, z int) []float64 {
	w := make([]float64, h)
	for y := a; y < z; y++ {
		w[y] = 1
	}
	for d := 1; d <= overlapRows; d++ {
		f := 0.5 * float64(overlapRows+1-d) / float64(overlapRows)
		if y := a - d; y >= 0 {
			w[y] = f
		}
		if y := z - 1 + d; y < h {
			w[y] = f
		}
	}
	return w
}

// bandedPlan builds one palette per contiguous band; band i lands in
// hardware palette i, top to bottom.
func bandedPlan(lb [][]lineBin, cuts []int) pipeline.Plan {
	var p pipeline.Plan
	h := len(lb)
	for b := 0; b < 16; b++ {
		a, z := cuts[b], cuts[b+1]
		p.Palettes[b] = paletteFromBins(binsFor(lb, bandWeights(h, a, z)))
		for y := a; y < z; y++ {
			p.Line[y] = uint8(b)
		}
	}
	return p
}

// adaptiveCuts chooses the 15 rows where the palette may switch by dynamic
// programming: minimize the total within-band weighted variance of pixel
// Oklab values. Cuts land where the image content already changes (an edge,
// a horizon) — exactly where a palette switch is least visible — and a band
// covering homogeneous content needs fewer distinct colors, so its 16
// entries go further.
func adaptiveCuts(lb [][]lineBin) []int {
	h := len(lb)
	// Prefix sums of weight, Oklab, and Oklab² per row make any band's
	// variance O(1): sse = Σx² − (Σx)²/n per axis.
	N := make([]float64, h+1)
	var S, Q [3][]float64
	for k := 0; k < 3; k++ {
		S[k] = make([]float64, h+1)
		Q[k] = make([]float64, h+1)
	}
	for y := 0; y < h; y++ {
		N[y+1] = N[y]
		for k := 0; k < 3; k++ {
			S[k][y+1] = S[k][y]
			Q[k][y+1] = Q[k][y]
		}
		for i := range lb[y] {
			b := &lb[y][i]
			N[y+1] += b.n
			for k, v := range [3]float64{b.L, b.A, b.B} {
				S[k][y+1] += b.n * v
				Q[k][y+1] += b.n * v * v
			}
		}
	}
	cost := func(a, z int) float64 {
		n := N[z] - N[a]
		if n == 0 {
			return 0
		}
		var t float64
		for k := 0; k < 3; k++ {
			s := S[k][z] - S[k][a]
			t += Q[k][z] - Q[k][a] - s*s/n
		}
		return t
	}

	const bands = 16
	// D[k][z] = least cost of covering rows [0,z) with k non-empty bands;
	// P[k][z] = start row of the k-th band in that optimum. Strict < keeps
	// the earliest split on ties, so the result is deterministic.
	D := make([][]float64, bands+1)
	P := make([][]int, bands+1)
	for k := range D {
		D[k] = make([]float64, h+1)
		P[k] = make([]int, h+1)
		for z := range D[k] {
			D[k][z] = math.Inf(1)
		}
	}
	for z := 1; z <= h; z++ {
		D[1][z] = cost(0, z)
	}
	for k := 2; k <= bands; k++ {
		for z := k; z <= h; z++ {
			for a := k - 1; a < z; a++ {
				if c := D[k-1][a] + cost(a, z); c < D[k][z] {
					D[k][z] = c
					P[k][z] = a
				}
			}
		}
	}

	cuts := make([]int, bands+1)
	cuts[bands] = h
	for k := bands; k >= 2; k-- {
		cuts[k-1] = P[k][cuts[k]]
	}
	return cuts // cuts[0] is 0
}

// perLinePlan is the fully dynamic mode: scanlines are clustered by content
// (k-means over lines) so any line may use any of the 16 palettes. Seeded
// from the adaptive contiguous cuts; each round rebuilds every cluster's
// palette from its member lines, then moves each line to the palette that
// reproduces it with the least Oklab error. A cluster that empties is
// reseeded with the worst-served line, so no palette sits idle while error
// remains — and the degenerate every-line-looks-alike start cannot collapse
// onto a single palette.
func perLinePlan(lb [][]lineBin) pipeline.Plan {
	h := len(lb)
	const clusters = 16
	assign := make([]int, h)
	var members [clusters]int
	cuts := adaptiveCuts(lb)
	for b := 0; b < clusters; b++ {
		for y := cuts[b]; y < cuts[b+1]; y++ {
			assign[y] = b
			members[b]++
		}
	}

	var pals [clusters][16]shr.RGB12
	var labs [clusters][16][3]float64
	// lineCost is line y's total squared Oklab error under cluster c's
	// palette: every bin matched to its nearest entry, weighted by pixels.
	lineCost := func(y, c int) float64 {
		var t float64
		for i := range lb[y] {
			b := &lb[y][i]
			best := math.Inf(1)
			for _, e := range labs[c] {
				dL, dA, dB := b.L-e[0], b.A-e[1], b.B-e[2]
				if d := dL*dL + dA*dA + dB*dB; d < best {
					best = d
				}
			}
			t += b.n * best
		}
		return t
	}

	weight := make([]float64, h)
	for iter := 0; iter < 16; iter++ {
		// Rebuild each non-empty cluster's palette from its member lines;
		// empty clusters keep their previous palette until reseeded.
		for c := 0; c < clusters; c++ {
			if members[c] == 0 {
				continue
			}
			for y := range weight {
				if assign[y] == c {
					weight[y] = 1
				} else {
					weight[y] = 0
				}
			}
			pals[c] = paletteFromBins(binsFor(lb, weight))
			for e, col := range pals[c] {
				labs[c][e][0], labs[c][e][1], labs[c][e][2] = pix.LinearToOklab(
					float64(pix.SRGBToLinear(float32(col.R)/15)),
					float64(pix.SRGBToLinear(float32(col.G)/15)),
					float64(pix.SRGBToLinear(float32(col.B)/15)),
				)
			}
		}

		// Move every line to the cluster whose palette serves it best.
		// First-wins on ties keeps the result deterministic.
		changed := false
		for y := 0; y < h; y++ {
			best, bestCost := 0, math.Inf(1)
			for c := 0; c < clusters; c++ {
				if d := lineCost(y, c); d < bestCost {
					bestCost = d
					best = c
				}
			}
			if best != assign[y] {
				members[assign[y]]--
				members[best]++
				assign[y] = best
				changed = true
			}
		}

		// Reseed: an empty cluster grabs the worst-served line from any
		// cluster that can spare one (ties: lowest y). Next round it gets
		// that line's exact palette, and similar lines can follow.
		for c := 0; c < clusters; c++ {
			if members[c] != 0 {
				continue
			}
			worst, worstCost := -1, 0.0
			for y := 0; y < h; y++ {
				if members[assign[y]] < 2 {
					continue // taking it would just empty another cluster
				}
				if d := lineCost(y, assign[y]); d > worstCost {
					worstCost = d
					worst = y
				}
			}
			if worst < 0 {
				break // every line is already served perfectly
			}
			members[assign[worst]]--
			assign[worst] = c
			members[c]++
			changed = true
		}
		if !changed {
			break
		}
	}

	// Relabel clusters in first-use order: palette 0 serves the top of the
	// screen, unused slots (never referenced by any SCB) sort to the end and
	// stay all-black zero palettes.
	var p pipeline.Plan
	var slot [clusters]int
	for c := range slot {
		slot[c] = -1
	}
	next := 0
	for y := 0; y < h; y++ {
		c := assign[y]
		if slot[c] < 0 {
			slot[c] = next
			p.Palettes[next] = pals[c]
			next++
		}
		p.Line[y] = uint8(slot[c])
	}
	return p
}
