// Package dither holds Ditherer implementations. Working: none (nearest
// color) and floyd-steinberg. The other names from the CLI are registered
// stubs so the flag parses today (see stubs.go).
//
// Palette matching is perceptual, error accounting is physical: nearest
// means smallest squared Euclidean distance in Oklab, while error vectors
// are computed and diffused in linear light. Linear-RGB matching was tried
// first and favors dark colors badly — mid-tones sit near black in linear
// space, so accumulated diffusion error flipped near-white pixels straight
// to black (speckle) instead of to a plausible neighbor.
package dither

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

// linearPalette is a hardware palette converted to linear light (for error
// vectors) with each entry's Oklab position cached (for matching).
type linearPalette struct {
	lin [16][3]float32
	lab [16][3]float64
}

func toLinear(pal [16]shr.RGB12) linearPalette {
	var lp linearPalette
	for i, c := range pal {
		lp.lin[i][0] = pix.SRGBToLinear(float32(c.R) / 15)
		lp.lin[i][1] = pix.SRGBToLinear(float32(c.G) / 15)
		lp.lin[i][2] = pix.SRGBToLinear(float32(c.B) / 15)
		lp.lab[i][0], lp.lab[i][1], lp.lab[i][2] = pix.LinearToOklab(
			float64(lp.lin[i][0]), float64(lp.lin[i][1]), float64(lp.lin[i][2]))
	}
	return lp
}

// nearest returns the palette index perceptually closest to linear (r,g,b)
// — smallest squared Oklab distance — and the error vector (pixel minus
// chosen color) in linear light for diffusion.
func (lp *linearPalette) nearest(r, g, b float32) (idx uint8, er, eg, eb float32) {
	L, A, B := pix.LinearToOklab(float64(r), float64(g), float64(b))
	best := -1.0
	for i, c := range lp.lab {
		dL, dA, dB := L-c[0], A-c[1], B-c[2]
		d := dL*dL + dA*dA + dB*dB
		if best < 0 || d < best {
			best = d
			idx = uint8(i)
		}
	}
	c := lp.lin[idx]
	return idx, r - c[0], g - c[1], b - c[2]
}

// check320 rejects work the v1 ditherers can't do yet. 640-mode dithering
// (choosing 2-bit values through the position-dependent palette groups) is
// a real algorithm the maintainer will spec later; guessing it here is an
// explicit anti-goal.
func check320(plan pipeline.Plan, src *pix.LinearImage) error {
	if plan.Mode640 {
		return fmt.Errorf("640-mode dithering: %w", pipeline.ErrNotImplemented)
	}
	if src.W != shr.Width320 || src.H != shr.Height {
		return fmt.Errorf("ditherer input is %dx%d, want %dx%d", src.W, src.H, shr.Width320, shr.Height)
	}
	return nil
}
