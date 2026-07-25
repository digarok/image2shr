package target

import (
	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/internal/planner"
	"github.com/digarok/image2shr/shr"
)

func init() { pipeline.RegisterTarget(color256{}) }

// color256: 320×200, up to 16 adaptive 16-color palettes with per-line SCB
// assignment. How the palettes are distributed over the scanlines is chosen
// by --scb-mode — see planner.AdaptiveColor256.
type color256 struct{}

func (color256) Name() string { return "shr320-color256" }

func (color256) Description() string {
	return "320x200 color, up to 16 adaptive 16-color palettes, per-line SCBs (--scb-mode)"
}

func (color256) Geometry() (w, h int) { return shr.Width320, shr.Height }

func (color256) Convert(src *pix.LinearImage, opt pipeline.Options) (*shr.Frame, error) {
	return pipeline.Run(src, planner.NewAdaptiveColor256(), opt)
}

// enforce interface compliance at compile time
var _ pipeline.Target = color256{}
