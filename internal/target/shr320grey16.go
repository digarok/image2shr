// Package target holds conversion targets — one file each, registered in
// init() so "image2shr targets" lists them and new ones are a single file.
package target

import (
	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/internal/planner"
	"github.com/digarok/image2shr/shr"
)

func init() { pipeline.RegisterTarget(grey16{}) }

// grey16: 320×200, single palette (hardware slot 0) on all 200 lines,
// 16 evenly spaced greys ($0000…$0FFF). All SCBs = $00.
type grey16 struct{}

func (grey16) Name() string { return "shr320-grey16" }

func (grey16) Description() string {
	return "320x200 greyscale, single 16-grey palette, all SCBs $00"
}

func (grey16) Geometry() (w, h int) { return shr.Width320, shr.Height }

func (grey16) Convert(src *pix.LinearImage, opt pipeline.Options) (*shr.Frame, error) {
	weights, err := pix.LumaOf(opt.Luma)
	if err != nil {
		return nil, err
	}
	// Reduce to luma first so the generic RGB-nearest ditherers quantize
	// against the grey ramp correctly for colored input.
	grey := pix.ToGrey(src, weights)
	return pipeline.Run(grey, planner.NewFixed("fixed-grey16", planner.Grey16Palette()), opt)
}

// enforce interface compliance at compile time
var _ pipeline.Target = grey16{}
