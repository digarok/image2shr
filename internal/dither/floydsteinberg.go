package dither

import (
	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
)

func init() { pipeline.RegisterDitherer(floydSteinberg{}) }

// floydSteinberg is classic Floyd–Steinberg error diffusion in linear
// light. Error distribution (× = current pixel, weights /16):
//
//	   ×  7
//	3  5  1
//
// --serpentine reverses the scan direction on alternate rows (mirroring the
// kernel); --dither-strength scales the diffused error (1.0 = full).
type floydSteinberg struct{}

func (floydSteinberg) Name() string { return "floyd-steinberg" }

func (floydSteinberg) Apply(src *pix.LinearImage, plan pipeline.Plan, opt pipeline.Options) (pipeline.Indexed, error) {
	if err := check320(plan, src); err != nil {
		return pipeline.Indexed{}, err
	}
	w, h := src.W, src.H
	out := pipeline.Indexed{W: w, H: h, Pix: make([]uint8, w*h)}
	strength := float32(opt.DitherStrength)

	// Working copy of the pixels; diffusion mutates neighbors.
	buf := make([]float32, len(src.Pix))
	copy(buf, src.Pix)

	var pals [16]*linearPalette
	// add spreads a share of the quantization error onto neighbor (x, y).
	add := func(x, y int, er, eg, eb, weight float32) {
		if x < 0 || x >= w || y >= h {
			return
		}
		i := 3 * (y*w + x)
		buf[i] += er * weight
		buf[i+1] += eg * weight
		buf[i+2] += eb * weight
	}

	for y := 0; y < h; y++ {
		pn := plan.Line[y]
		if pals[pn] == nil {
			lp := toLinear(plan.Palettes[pn])
			pals[pn] = &lp
		}
		lp := pals[pn]

		// dir = +1 left→right; serpentine flips odd rows and mirrors the
		// kernel horizontally.
		dir := 1
		x0 := 0
		if opt.Serpentine && y%2 == 1 {
			dir = -1
			x0 = w - 1
		}
		for x := x0; x >= 0 && x < w; x += dir {
			i := 3 * (y*w + x)
			r := pix.Clamp01(buf[i])
			g := pix.Clamp01(buf[i+1])
			b := pix.Clamp01(buf[i+2])
			idx, er, eg, eb := lp.nearest(r, g, b)
			out.Pix[y*w+x] = idx

			er *= strength
			eg *= strength
			eb *= strength
			add(x+dir, y, er, eg, eb, 7.0/16)
			add(x-dir, y+1, er, eg, eb, 3.0/16)
			add(x, y+1, er, eg, eb, 5.0/16)
			add(x+dir, y+1, er, eg, eb, 1.0/16)
		}
	}
	return out, nil
}
