// Package dither holds Ditherer implementations. Working: none (nearest
// color) and floyd-steinberg. The other names from the CLI are registered
// stubs so the flag parses today (see stubs.go).
//
// All matching happens in linear light: palette entries are converted from
// their 4-bit sRGB-style values to linear RGB, and nearest means smallest
// squared distance in that space. Deliberately the dumbest correct metric.
package dither

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

// linearPalette is a hardware palette converted to linear light for
// distance matching. RGB12 channel n displays as sRGB n/15.
type linearPalette [16][3]float32

func toLinear(pal [16]shr.RGB12) linearPalette {
	var lp linearPalette
	for i, c := range pal {
		lp[i][0] = pix.SRGBToLinear(float32(c.R) / 15)
		lp[i][1] = pix.SRGBToLinear(float32(c.G) / 15)
		lp[i][2] = pix.SRGBToLinear(float32(c.B) / 15)
	}
	return lp
}

// nearest returns the palette index closest to (r,g,b) and the error vector
// (pixel minus chosen color).
func (lp *linearPalette) nearest(r, g, b float32) (idx uint8, er, eg, eb float32) {
	best := float32(-1)
	for i, c := range lp {
		dr, dg, db := r-c[0], g-c[1], b-c[2]
		d := dr*dr + dg*dg + db*db
		if best < 0 || d < best {
			best = d
			idx = uint8(i)
		}
	}
	c := lp[idx]
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
