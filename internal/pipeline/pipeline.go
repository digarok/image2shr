package pipeline

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

// Plan is a PalettePlanner's output: the 16 hardware palettes and which
// palette each of the 200 scanlines uses. Milestone 1 uses one palette and
// one mode for the whole frame; the per-line fields exist so multi-SCB
// planners plug in without changing the seams.
type Plan struct {
	Palettes [16][16]shr.RGB12
	Line     [200]uint8 // palette number (0-15) per scanline
	Mode640  bool       // per-frame for now; SCBs are still written per line
	Fill     bool       // 320-mode color fill (v1 planners never set it)
}

// Indexed is a Ditherer's output: one raw pixel VALUE per pixel, exactly as
// it will be packed — a 4-bit palette index in 320 mode, a 2-bit value in
// 640 mode. In 640 mode a value's meaning depends on x: resolve through
// shr.PaletteIndex640, never by indexing the palette directly.
type Indexed struct {
	W, H int
	Pix  []uint8
}

// The three pipeline stage interfaces. Implementations register themselves
// in init() and are looked up by flag value.

type PalettePlanner interface {
	Name() string
	Plan(src *pix.LinearImage, opt Options) (Plan, error)
}

type Ditherer interface {
	Name() string
	Apply(src *pix.LinearImage, plan Plan, opt Options) (Indexed, error)
}

type Target interface {
	Name() string // e.g. "shr320-grey16"
	Description() string
	Geometry() (w, h int)
	Convert(src *pix.LinearImage, opt Options) (*shr.Frame, error)
}

// Run is the shared plan → dither → pack glue. Targets call it with their
// planner of choice; the ditherer comes from the flags.
func Run(src *pix.LinearImage, planner PalettePlanner, opt Options) (*shr.Frame, error) {
	d, err := LookupDitherer(opt.Dither)
	if err != nil {
		return nil, err
	}
	plan, err := planner.Plan(src, opt)
	if err != nil {
		return nil, fmt.Errorf("planner %s: %w", planner.Name(), err)
	}
	idx, err := d.Apply(src, plan, opt)
	if err != nil {
		return nil, fmt.Errorf("ditherer %s: %w", d.Name(), err)
	}
	return PackFrame(plan, idx)
}

// PackFrame packs dithered pixel values plus the plan into a Frame: pixel
// bytes, one SCB per scanline, and the palettes.
func PackFrame(plan Plan, idx Indexed) (*shr.Frame, error) {
	wantW := shr.Width320
	if plan.Mode640 {
		wantW = shr.Width640
	}
	if idx.W != wantW || idx.H != shr.Height {
		return nil, fmt.Errorf("packing: indexed image is %dx%d, want %dx%d", idx.W, idx.H, wantW, shr.Height)
	}
	f := &shr.Frame{Palettes: plan.Palettes}
	for y := 0; y < shr.Height; y++ {
		f.SCB[y] = shr.MakeSCB(plan.Mode640, false, plan.Fill, plan.Line[y])
		row := idx.Pix[y*idx.W : (y+1)*idx.W]
		var err error
		if plan.Mode640 {
			err = shr.PackLine640(f.Line(y), row)
		} else {
			err = shr.PackLine320(f.Line(y), row)
		}
		if err != nil {
			return nil, fmt.Errorf("packing line %d: %w", y, err)
		}
	}
	return f, nil
}
