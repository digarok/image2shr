package target

import (
	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/internal/planner"
	"github.com/digarok/image2shr/shr"
)

func init() { pipeline.RegisterTarget(color3200{}) }

// color3200: 320×200, one adaptive 16-color palette per scanline — the
// Brooks 3200-color format. The frame cannot fit the 32KB raw container, so
// --format resolves to brooks for it.
type color3200 struct{}

func (color3200) Name() string { return "shr320-color3200" }

func (color3200) Description() string {
	return "320x200 color, one adaptive 16-color palette per scanline (Brooks 3200)"
}

func (color3200) Geometry() (w, h int) { return shr.Width320, shr.Height }

func (color3200) Convert(src *pix.LinearImage, opt pipeline.Options) (*shr.Frame, error) {
	return pipeline.Run(src, planner.NewAdaptiveColor3200(), opt)
}

// enforce interface compliance at compile time
var _ pipeline.Target = color3200{}
