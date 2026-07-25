package target

import (
	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/internal/planner"
	"github.com/digarok/image2shr/shr"
)

func init() { pipeline.RegisterTarget(color16{}) }

// color16: 320×200, one adaptive 16-color palette in hardware slot 0 on all
// 200 lines. The palette is chosen by perceptual (Oklab) clustering of the
// prepared image — see planner.AdaptiveColor16. All SCBs = $00.
type color16 struct{}

func (color16) Name() string { return "shr320-color16" }

func (color16) Description() string {
	return "320x200 color, single adaptive 16-color palette, all SCBs $00"
}

func (color16) Geometry() (w, h int) { return shr.Width320, shr.Height }

func (color16) Convert(src *pix.LinearImage, opt pipeline.Options) (*shr.Frame, error) {
	return pipeline.Run(src, planner.NewAdaptiveColor16(), opt)
}

// enforce interface compliance at compile time
var _ pipeline.Target = color16{}
