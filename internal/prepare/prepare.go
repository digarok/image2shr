// Package prepare implements the pipeline's image preparation stage: crop,
// fit/scale with pixel-aspect-ratio handling, and tone adjustments. All
// operations work on pix.LinearImage (linear light).
package prepare

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pix"
)

// Fit mode names, matching the --fit flag values.
const (
	FitContain = "contain" // scale to fit entirely, letterbox with black
	FitCover   = "cover"   // scale to fill, crop overflow
	FitStretch = "stretch" // scale each axis independently
	FitNone    = "none"    // 1 source pixel = 1 SHR pixel, centered, no scaling
)

// Crop extracts the rectangle (x, y, w, h) in source-image coordinates.
func Crop(src *pix.LinearImage, x, y, w, h int) (*pix.LinearImage, error) {
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("crop size %dx%d must be positive", w, h)
	}
	if x < 0 || y < 0 || x+w > src.W || y+h > src.H {
		return nil, fmt.Errorf("crop %d,%d,%d,%d outside source bounds %dx%d", x, y, w, h, src.W, src.H)
	}
	out := pix.New(w, h)
	for row := 0; row < h; row++ {
		srcOff := 3 * ((y+row)*src.W + x)
		dstOff := 3 * row * w
		copy(out.Pix[dstOff:dstOff+3*w], src.Pix[srcOff:srcOff+3*w])
	}
	return out, nil
}

// Resample scales src onto a dstW×dstH framebuffer.
//
// par is the pixel aspect ratio of the DESTINATION pixels: display height of
// one pixel divided by its width. SHR pixels are not square — on the 4:3
// screen a 320-mode pixel is ~1.2× taller than wide and a 640-mode pixel
// ~2.4× (par = (w/h) * 3/4 for a w×h SHR framebuffer). Pass par = 1.0 to
// ignore aspect (--aspect ignore).
//
// Internally the destination is modeled in "display units" (dstW × dstH*par)
// so the source's proportions survive onto non-square pixels; each
// destination pixel is then an area-average (box filter) of the source
// region it covers. The box filter is the deliberately simple v1 resampler.
// Areas not covered by the source (letterboxing) are black.
func Resample(src *pix.LinearImage, dstW, dstH int, par float64, fit string) (*pix.LinearImage, error) {
	out := pix.New(dstW, dstH)

	if fit == FitNone {
		// 1:1 pixel mapping, centered; par is irrelevant since nothing scales.
		offX := (dstW - src.W) / 2
		offY := (dstH - src.H) / 2
		for y := 0; y < dstH; y++ {
			sy := y - offY
			if sy < 0 || sy >= src.H {
				continue
			}
			for x := 0; x < dstW; x++ {
				sx := x - offX
				if sx < 0 || sx >= src.W {
					continue
				}
				r, g, b := src.At(sx, sy)
				out.Set(x, y, r, g, b)
			}
		}
		return out, nil
	}

	// Display-unit size of the destination.
	dispW := float64(dstW)
	dispH := float64(dstH) * par

	// Placement of the source image in display units: source rect maps to
	// [offX, offX+drawW) × [offY, offY+drawH).
	var offX, offY, drawW, drawH float64
	switch fit {
	case FitStretch:
		drawW, drawH = dispW, dispH
	case FitContain, FitCover:
		sx := dispW / float64(src.W)
		sy := dispH / float64(src.H)
		s := min(sx, sy)
		if fit == FitCover {
			s = max(sx, sy)
		}
		drawW = float64(src.W) * s
		drawH = float64(src.H) * s
		offX = (dispW - drawW) / 2
		offY = (dispH - drawH) / 2
	default:
		return nil, fmt.Errorf("unknown fit mode %q (valid: contain, cover, stretch, none)", fit)
	}

	// For each destination pixel, find the source region it covers and
	// box-average it.
	for y := 0; y < dstH; y++ {
		// This pixel's span in display units, then in source coordinates.
		dy0 := float64(y) * par
		dy1 := float64(y+1) * par
		sy0 := (dy0 - offY) / drawH * float64(src.H)
		sy1 := (dy1 - offY) / drawH * float64(src.H)
		for x := 0; x < dstW; x++ {
			sx0 := (float64(x) - offX) / drawW * float64(src.W)
			sx1 := (float64(x+1) - offX) / drawW * float64(src.W)
			r, g, b, ok := boxSample(src, sx0, sy0, sx1, sy1)
			if ok {
				out.Set(x, y, r, g, b)
			}
		}
	}
	return out, nil
}

// boxSample averages src over the rectangle [x0,x1)×[y0,y1) in source pixel
// coordinates, weighting edge pixels by fractional coverage. Returns
// ok=false when the rectangle lies entirely outside the source (letterbox —
// leave black).
func boxSample(src *pix.LinearImage, x0, y0, x1, y1 float64) (r, g, b float32, ok bool) {
	x0 = max(x0, 0)
	y0 = max(y0, 0)
	x1 = min(x1, float64(src.W))
	y1 = min(y1, float64(src.H))
	if x1 <= x0 || y1 <= y0 {
		return 0, 0, 0, false
	}

	var sr, sg, sb, total float64
	for iy := int(y0); float64(iy) < y1; iy++ {
		wy := min(y1, float64(iy+1)) - max(y0, float64(iy))
		for ix := int(x0); float64(ix) < x1; ix++ {
			wx := min(x1, float64(ix+1)) - max(x0, float64(ix))
			w := wx * wy
			pr, pg, pb := src.At(ix, iy)
			sr += float64(pr) * w
			sg += float64(pg) * w
			sb += float64(pb) * w
			total += w
		}
	}
	if total == 0 {
		return 0, 0, 0, false
	}
	return float32(sr / total), float32(sg / total), float32(sb / total), true
}
