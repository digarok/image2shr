// Package planner holds PalettePlanner implementations. Milestone 1 has
// exactly one: a fixed palette on all 200 lines. Real planners (per-line,
// grouped — the multi-SCB problem) plug in here later.
package planner

import (
	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

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

func (f *Fixed) Plan(_ *pix.LinearImage, _ pipeline.Options) (pipeline.Plan, error) {
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
