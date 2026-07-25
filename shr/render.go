package shr

import (
	"image"
	"image/color"
)

// Render turns a Frame into the RGB image an Apple IIgs would display.
// It is the correctness anchor for the whole project: honors per-line mode,
// per-line palette, the 640-mode pixel-position→palette-group mapping, and
// 320-mode color fill.
//
// The canvas is always 640×200 so frames that mix 320- and 640-mode
// scanlines render onto one image: a 320-mode pixel is drawn as two adjacent
// canvas pixels. No aspect-ratio correction is applied here — SHR pixels are
// not square (see docs/SPEC.md); scaling for display is the caller's job.
func Render(f *Frame) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, Width640, Height))
	for y := 0; y < Height; y++ {
		scb := f.SCB[y]
		pal := &f.Palettes[SCBPalette(scb)]
		row := f.Line(y)

		if SCBIs640(scb) {
			vals := UnpackLine640(row)
			for x, v := range vals {
				setPixel(img, x, y, pal[PaletteIndex640(x, v)].Color())
			}
			continue
		}

		// 320 mode: each pixel drawn 2 canvas pixels wide.
		idx := UnpackLine320(row)
		fill := SCBIsFill(scb)
		// Color fill mode: pixel value 0 repeats the previous pixel's color
		// instead of meaning "palette entry 0". Hardware behavior is
		// undefined when the leftmost pixel is 0; we render palette entry 0.
		prev := pal[0].Color()
		for x, v := range idx {
			var c color.RGBA
			if fill && v == 0 && x > 0 {
				c = prev
			} else {
				c = pal[v].Color()
			}
			prev = c
			setPixel(img, 2*x, y, c)
			setPixel(img, 2*x+1, y, c)
		}
	}
	return img
}

func setPixel(img *image.RGBA, x, y int, c color.RGBA) {
	off := img.PixOffset(x, y)
	img.Pix[off] = c.R
	img.Pix[off+1] = c.G
	img.Pix[off+2] = c.B
	img.Pix[off+3] = c.A
}
