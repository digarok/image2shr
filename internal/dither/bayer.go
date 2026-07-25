package dither

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
)

func init() {
	for _, n := range []int{2, 4, 8} {
		pipeline.RegisterDitherer(newBayer(n))
	}
}

// bayer is ordered dithering with an n×n Bayer threshold matrix
// (bayer2/4/8). Each pixel is nudged by a position-dependent threshold
// before nearest-color matching; no error diffusion, so the pattern is a
// fixed n×n tile — the retro "crosshatch" look, and stable across
// animation frames.
//
// The nudge is applied per channel in sRGB space with a spread of 1/15 —
// one step of the 4-bit hardware channel. Against a uniform ramp like the
// grey16 palette this is exactly classic ordered dithering; for arbitrary
// palettes it is the standard naive approximation (threshold-then-nearest),
// kept deliberately simple. --dither-strength scales the nudge;
// --serpentine has no effect (there is no scan order).
type bayer struct {
	name string
	n    int
	// thresholds[y%n][x%n], normalized to (-0.5, 0.5).
	thresholds [][]float32
}

func newBayer(n int) bayer {
	m := bayerMatrix(n)
	t := make([][]float32, n)
	for y := range t {
		t[y] = make([]float32, n)
		for x := range t[y] {
			t[y][x] = (float32(m[y][x])+0.5)/float32(n*n) - 0.5
		}
	}
	return bayer{name: fmt.Sprintf("bayer%d", n), n: n, thresholds: t}
}

// bayerMatrix builds the standard recursive Bayer index matrix: M2 is
//
//	0 2
//	3 1
//
// and each doubling replicates the previous matrix ×4 with quadrant
// offsets 0 (top-left), 2 (top-right), 3 (bottom-left), 1 (bottom-right).
func bayerMatrix(n int) [][]int {
	m := [][]int{{0, 2}, {3, 1}}
	for len(m) < n {
		h := len(m)
		next := make([][]int, 2*h)
		for y := range next {
			next[y] = make([]int, 2*h)
		}
		for y := 0; y < h; y++ {
			for x := 0; x < h; x++ {
				v := 4 * m[y][x]
				next[y][x] = v
				next[y][x+h] = v + 2
				next[y+h][x] = v + 3
				next[y+h][x+h] = v + 1
			}
		}
		m = next
	}
	return m
}

func (b bayer) Name() string { return b.name }

func (b bayer) Apply(src *pix.LinearImage, plan pipeline.Plan, opt pipeline.Options) (pipeline.Indexed, error) {
	if err := check320(plan, src); err != nil {
		return pipeline.Indexed{}, err
	}
	out := pipeline.Indexed{W: src.W, H: src.H, Pix: make([]uint8, src.W*src.H)}
	strength := float32(opt.DitherStrength)
	pc := palCache{plan: &plan}
	for y := 0; y < src.H; y++ {
		lp := pc.forLine(y)
		row := b.thresholds[y%b.n]
		for x := 0; x < src.W; x++ {
			r, g, bl := src.At(x, y)
			// Nudge in sRGB by threshold × one 4-bit channel step.
			d := row[x%b.n] * strength / 15
			r = pix.SRGBToLinear(pix.Clamp01(pix.LinearToSRGB(r) + d))
			g = pix.SRGBToLinear(pix.Clamp01(pix.LinearToSRGB(g) + d))
			bl = pix.SRGBToLinear(pix.Clamp01(pix.LinearToSRGB(bl) + d))
			idx, _, _, _ := lp.nearest(r, g, bl)
			out.Pix[y*src.W+x] = idx
		}
	}
	return out, nil
}
