package dither

import (
	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
)

// tap is one entry of an error-diffusion kernel: the share (weight) of a
// pixel's quantization error pushed onto the neighbor at (dx, dy). dx is
// relative to the scan direction — serpentine rows mirror it.
type tap struct {
	dx, dy int
	weight float32
}

// diffuser is the shared engine behind every error-diffusion ditherer;
// the algorithms differ only in their kernel. Scan is left→right
// (--serpentine reverses odd rows and mirrors the kernel); each pixel is
// matched to the nearest palette color, then the error — scaled by
// --dither-strength — is spread over the taps in linear light.
type diffuser struct {
	name   string
	kernel []tap
}

func (d diffuser) Name() string { return d.name }

func (d diffuser) Apply(src *pix.LinearImage, plan pipeline.Plan, opt pipeline.Options) (pipeline.Indexed, error) {
	if err := check320(plan, src); err != nil {
		return pipeline.Indexed{}, err
	}
	w, h := src.W, src.H
	out := pipeline.Indexed{W: w, H: h, Pix: make([]uint8, w*h)}
	strength := float32(opt.DitherStrength)

	// Working copy of the pixels; diffusion mutates neighbors.
	buf := make([]float32, len(src.Pix))
	copy(buf, src.Pix)

	pc := palCache{plan: &plan}
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
		lp := pc.forLine(y)

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
			for _, t := range d.kernel {
				add(x+t.dx*dir, y+t.dy, er, eg, eb, t.weight)
			}
		}
	}
	return out, nil
}
