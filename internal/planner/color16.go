package planner

import (
	"math"
	"sort"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

// AdaptiveColor16 picks the 16 hardware colors that reproduce the source
// with the least perceived error and assigns them as one palette in slot 0
// for all 200 scanlines, 320 mode.
//
// The reduction is psychovisual, not a popularity histogram: pixels are
// clustered by Euclidean distance in Oklab, a perceptually uniform space,
// so a small but visually distinct region (a red taillight in a grey
// street) earns a palette entry that raw pixel counts would never give it.
// Initial clusters come from variance-based median cut, refined by weighted
// k-means; every loop runs in a fixed order, so the result is deterministic.
type AdaptiveColor16 struct{}

// NewAdaptiveColor16 returns the adaptive single-palette color planner.
func NewAdaptiveColor16() *AdaptiveColor16 { return &AdaptiveColor16{} }

func (*AdaptiveColor16) Name() string { return "adaptive-color16" }

func (a *AdaptiveColor16) Plan(src *pix.LinearImage, opt pipeline.Options) (pipeline.Plan, error) {
	if err := requireSinglePalette(a.Name(), opt); err != nil {
		return pipeline.Plan{}, err
	}
	var p pipeline.Plan
	p.Palettes[0] = bestPalette16(src)
	// p.Line stays all zeros: every scanline uses palette 0.
	return p, nil
}

// cbin aggregates the source pixels sharing one RGB444 cell. The hardware
// palette cannot distinguish finer than 4 bits per channel, so clustering
// at most 4096 weighted bins loses nothing that could reach the screen and
// keeps the cost independent of image size.
type cbin struct {
	r, g, b float64 // mean linear RGB of the member pixels
	n       float64 // pixel count (cluster weight)
	L, A, B float64 // Oklab of the mean color, cached for distance tests
}

// centroid is a cluster's running mean, kept in linear RGB (physically
// meaningful averaging) with its Oklab position cached for matching.
type centroid struct {
	r, g, b float64
	L, A, B float64
}

// bestPalette16 reduces src to at most 16 RGB444 colors, ordered dark to
// light. Unused trailing entries duplicate entry 0 so error diffusion can
// never be attracted to a color the image does not contain.
func bestPalette16(src *pix.LinearImage) [16]shr.RGB12 {
	return paletteFromBins(histogram444(src))
}

// paletteFromBins runs the reduction (median cut → k-means → RGB444 snap →
// dark-to-light sort) on prebuilt weighted bins, so multi-palette planners
// can feed it any weighted subset of the image. Empty input yields the
// all-black zero palette.
func paletteFromBins(bins []cbin) [16]shr.RGB12 {
	if len(bins) == 0 {
		return [16]shr.RGB12{}
	}
	groups := medianCut(bins, 16)
	cents := kmeans(bins, groups)

	type entry struct {
		c shr.RGB12
		l float64 // Oklab lightness of the snapped color, for stable ordering
	}
	entries := make([]entry, len(cents))
	for i, c := range cents {
		rgb := shr.RGB12{R: nib(c.r), G: nib(c.g), B: nib(c.b)}
		l, _, _ := pix.LinearToOklab(
			float64(pix.SRGBToLinear(float32(rgb.R)/15)),
			float64(pix.SRGBToLinear(float32(rgb.G)/15)),
			float64(pix.SRGBToLinear(float32(rgb.B)/15)),
		)
		entries[i] = entry{c: rgb, l: l}
	}
	sort.Slice(entries, func(a, b int) bool {
		ea, eb := entries[a], entries[b]
		if ea.l != eb.l {
			return ea.l < eb.l
		}
		ka := int(ea.c.R)<<8 | int(ea.c.G)<<4 | int(ea.c.B)
		kb := int(eb.c.R)<<8 | int(eb.c.G)<<4 | int(eb.c.B)
		return ka < kb // total order keeps the palette deterministic
	})

	var pal [16]shr.RGB12
	for i, e := range entries {
		pal[i] = e.c
	}
	for i := len(entries); i < 16; i++ {
		pal[i] = pal[0]
	}
	return pal
}

// nib converts a linear channel value to its 4-bit sRGB-encoded nibble.
func nib(v float64) uint8 {
	n := int(pix.Clamp01(pix.LinearToSRGB(float32(v)))*15 + 0.5)
	return uint8(n)
}

// histogram444 buckets every pixel into its RGB444 cell, keeping the mean
// linear color per cell. Cells are scanned in ascending key order, so the
// bin list (and everything built from it) is deterministic.
func histogram444(src *pix.LinearImage) []cbin {
	var cells [4096]cbin
	for i := 0; i < src.W*src.H; i++ {
		r := float64(src.Pix[3*i])
		g := float64(src.Pix[3*i+1])
		b := float64(src.Pix[3*i+2])
		key := int(nib(r))<<8 | int(nib(g))<<4 | int(nib(b))
		c := &cells[key]
		c.r += r
		c.g += g
		c.b += b
		c.n++
	}
	bins := make([]cbin, 0, 256)
	for key := range cells {
		c := cells[key]
		if c.n == 0 {
			continue
		}
		c.r /= c.n
		c.g /= c.n
		c.b /= c.n
		c.L, c.A, c.B = pix.LinearToOklab(c.r, c.g, c.b)
		bins = append(bins, c)
	}
	return bins
}

func axisVal(c *cbin, axis int) float64 {
	switch axis {
	case 0:
		return c.L
	case 1:
		return c.A
	default:
		return c.B
	}
}

// box is a median-cut cluster: a set of bins plus its weighted squared
// deviation from the cluster mean along each Oklab axis.
type box struct {
	bins []int
	sse  [3]float64
}

func (bx *box) stats(bins []cbin) {
	var n float64
	var mean [3]float64
	for _, i := range bx.bins {
		c := &bins[i]
		n += c.n
		mean[0] += c.n * c.L
		mean[1] += c.n * c.A
		mean[2] += c.n * c.B
	}
	for k := range mean {
		mean[k] /= n
	}
	bx.sse = [3]float64{}
	for _, i := range bx.bins {
		c := &bins[i]
		d := [3]float64{c.L - mean[0], c.A - mean[1], c.B - mean[2]}
		for k := range d {
			bx.sse[k] += c.n * d[k] * d[k]
		}
	}
}

func (bx *box) totalSSE() float64 { return bx.sse[0] + bx.sse[1] + bx.sse[2] }

func (bx *box) widestAxis() int {
	axis := 0
	for k := 1; k < 3; k++ {
		if bx.sse[k] > bx.sse[axis] {
			axis = k
		}
	}
	return axis
}

// medianCut splits the bins into at most k groups. Each step splits the
// group with the largest perceived error (weighted Oklab SSE) along its
// worst axis at the weighted median — so splits chase where the eye would
// see the most damage, not where the most pixels are.
func medianCut(bins []cbin, k int) [][]int {
	first := &box{bins: make([]int, len(bins))}
	for i := range first.bins {
		first.bins[i] = i
	}
	first.stats(bins)
	boxes := []*box{first}

	for len(boxes) < k {
		best := -1
		for i, bx := range boxes {
			if len(bx.bins) < 2 || bx.totalSSE() == 0 {
				continue // nothing left to separate
			}
			if best < 0 || bx.totalSSE() > boxes[best].totalSSE() {
				best = i
			}
		}
		if best < 0 {
			break // fewer distinct colors than palette entries
		}
		bx := boxes[best]
		axis := bx.widestAxis()
		sort.Slice(bx.bins, func(a, b int) bool {
			va := axisVal(&bins[bx.bins[a]], axis)
			vb := axisVal(&bins[bx.bins[b]], axis)
			if va != vb {
				return va < vb
			}
			return bx.bins[a] < bx.bins[b] // total order → deterministic
		})

		var total float64
		for _, i := range bx.bins {
			total += bins[i].n
		}
		cut, acc := 1, bins[bx.bins[0]].n
		for cut < len(bx.bins)-1 && acc < total/2 {
			acc += bins[bx.bins[cut]].n
			cut++
		}
		left := &box{bins: bx.bins[:cut:cut]}
		right := &box{bins: bx.bins[cut:]}
		left.stats(bins)
		right.stats(bins)
		boxes[best] = left
		boxes = append(boxes, right)
	}

	out := make([][]int, len(boxes))
	for i, bx := range boxes {
		out[i] = bx.bins
	}
	return out
}

// kmeans refines the median-cut groups: bins are reassigned to their
// perceptually nearest centroid until stable (or 32 rounds). Ties keep the
// lowest centroid index and empty clusters keep their previous position,
// both for determinism.
func kmeans(bins []cbin, groups [][]int) []centroid {
	cents := make([]centroid, len(groups))
	assign := make([]int, len(bins))
	for gi, g := range groups {
		for _, bi := range g {
			assign[bi] = gi
		}
	}

	recompute := func() {
		type sum struct{ r, g, b, n float64 }
		sums := make([]sum, len(cents))
		for bi := range bins {
			c := &bins[bi]
			s := &sums[assign[bi]]
			s.r += c.n * c.r
			s.g += c.n * c.g
			s.b += c.n * c.b
			s.n += c.n
		}
		for ci := range cents {
			if sums[ci].n == 0 {
				continue
			}
			cents[ci].r = sums[ci].r / sums[ci].n
			cents[ci].g = sums[ci].g / sums[ci].n
			cents[ci].b = sums[ci].b / sums[ci].n
			cents[ci].L, cents[ci].A, cents[ci].B =
				pix.LinearToOklab(cents[ci].r, cents[ci].g, cents[ci].b)
		}
	}
	recompute()

	for iter := 0; iter < 32; iter++ {
		changed := false
		for bi := range bins {
			c := &bins[bi]
			best, bestD := 0, math.Inf(1)
			for ci := range cents {
				dL := c.L - cents[ci].L
				dA := c.A - cents[ci].A
				dB := c.B - cents[ci].B
				if d := dL*dL + dA*dA + dB*dB; d < bestD {
					bestD = d
					best = ci
				}
			}
			if assign[bi] != best {
				assign[bi] = best
				changed = true
			}
		}
		if !changed {
			break
		}
		recompute()
	}
	return cents
}
