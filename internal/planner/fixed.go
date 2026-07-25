// Package planner holds PalettePlanner implementations: Fixed (a
// predetermined palette on all 200 lines), AdaptiveColor16 (a perceptual
// best-16-colors reduction, one palette for the whole screen), and
// AdaptiveColor256 (up to 16 palettes distributed over the scanlines — the
// multi-SCB planner, with banded / grouped / per-line strategies selected
// by --scb-mode).
package planner

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

// requireSinglePalette rejects multi-palette --scb-mode values for planners
// that produce one palette by construction ("auto" resolves to single here).
func requireSinglePalette(plannerName string, opt pipeline.Options) error {
	switch opt.SCBMode {
	case "", "auto", "single":
		return nil
	}
	return fmt.Errorf("planner %s always uses a single palette; --scb-mode %s needs a multi-palette target like shr320-color256",
		plannerName, opt.SCBMode)
}

// Fixed is the dumbest correct planner: one predetermined palette in
// hardware slot 0, every scanline assigned to it, 320 mode.
type Fixed struct {
	name    string
	palette [16]shr.RGB12
}

// NewFixed returns a planner that always yields the given palette.
func NewFixed(name string, palette [16]shr.RGB12) *Fixed {
	return &Fixed{name: name, palette: palette}
}

func (f *Fixed) Name() string { return f.name }

func (f *Fixed) Plan(_ *pix.LinearImage, opt pipeline.Options) (pipeline.Plan, error) {
	if err := requireSinglePalette(f.name, opt); err != nil {
		return pipeline.Plan{}, err
	}
	var p pipeline.Plan
	p.Palettes[0] = f.palette
	// p.Line is already all zeros: every scanline uses palette 0.
	return p, nil
}

// Grey16Palette is the milestone-1 palette: 16 evenly spaced greys,
// entries 0–15 = $0000, $0111, … $0FFF, i.e. RGB12{n,n,n}.
func Grey16Palette() [16]shr.RGB12 {
	var pal [16]shr.RGB12
	for n := uint8(0); n < 16; n++ {
		pal[n] = shr.RGB12{R: n, G: n, B: n}
	}
	return pal
}
