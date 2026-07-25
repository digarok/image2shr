package dither

import (
	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
)

func init() { pipeline.RegisterDitherer(none{}) }

// none is plain nearest-color quantization: no error diffusion at all.
type none struct{}

func (none) Name() string { return "none" }

func (none) Apply(src *pix.LinearImage, plan pipeline.Plan, _ pipeline.Options) (pipeline.Indexed, error) {
	if err := check320(plan, src); err != nil {
		return pipeline.Indexed{}, err
	}
	out := pipeline.Indexed{W: src.W, H: src.H, Pix: make([]uint8, src.W*src.H)}
	pc := palCache{plan: &plan}
	for y := 0; y < src.H; y++ {
		lp := pc.forLine(y)
		for x := 0; x < src.W; x++ {
			r, g, b := src.At(x, y)
			idx, _, _, _ := lp.nearest(r, g, b)
			out.Pix[y*src.W+x] = idx
		}
	}
	return out, nil
}
