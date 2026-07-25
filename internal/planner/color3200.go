package planner

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

// AdaptiveColor3200 gives every one of the 200 scanlines its own 16-color
// palette — the Brooks 3200-color trick, where a display routine reloads
// the color tables while racing the beam. Up to 3200 simultaneous colors;
// the hard 16-colors-per-line limit still applies, per line.
//
// Each line's palette is built from a sliding window: the line itself at
// full weight plus a few rows above and below at tapering weight. The
// window matters twice over. It keeps adjacent palettes strongly
// correlated, so the hue set drifts smoothly down the screen instead of
// flickering line to line; and it means the error Floyd–Steinberg diffuses
// into the next line lands on a palette that still contains similar colors
// to absorb it.
type AdaptiveColor3200 struct{}

// NewAdaptiveColor3200 returns the per-scanline-palette planner.
func NewAdaptiveColor3200() *AdaptiveColor3200 { return &AdaptiveColor3200{} }

func (*AdaptiveColor3200) Name() string { return "adaptive-color3200" }

func (a *AdaptiveColor3200) Plan(src *pix.LinearImage, opt pipeline.Options) (pipeline.Plan, error) {
	if src.W != shr.Width320 || src.H != shr.Height {
		return pipeline.Plan{}, fmt.Errorf("planner input is %dx%d, want %dx%d",
			src.W, src.H, shr.Width320, shr.Height)
	}
	switch opt.SCBMode {
	case "", "auto":
	default:
		return pipeline.Plan{}, fmt.Errorf(
			"--scb-mode %s: a 3200-color image has no SCBs — every line already gets its own palette", opt.SCBMode)
	}

	lb := lineBinsOf(src)
	var p pipeline.Plan
	p.LinePalettes = &[shr.Height][16]shr.RGB12{}
	weight := make([]float64, src.H)
	for y := 0; y < src.H; y++ {
		windowWeights(weight, y)
		p.LinePalettes[y] = paletteFromBins(binsFor(lb, weight))
	}
	// p.Palettes, p.Line, Mode640, Fill all stay zero: Brooks frames carry
	// no SCB table and are 320-mode by definition.
	return p, nil
}

// windowRows is how far the per-line palette training window reaches above
// and below the line. Neighbor rows taper from half the line's own weight
// to near zero, the same shape the color256 band overlap uses.
const windowRows = 3

func windowWeights(w []float64, y int) {
	for i := range w {
		w[i] = 0
	}
	w[y] = 1
	for d := 1; d <= windowRows; d++ {
		f := 0.5 * float64(windowRows+1-d) / float64(windowRows)
		if i := y - d; i >= 0 {
			w[i] = f
		}
		if i := y + d; i < len(w) {
			w[i] = f
		}
	}
}
