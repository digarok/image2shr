package shr

import (
	"errors"
	"image"
	"image/color"
)

// Uses640 reports whether any scanline's SCB selects 640 mode.
func (f *Frame) Uses640() bool {
	for y := 0; y < Height; y++ {
		if SCBIs640(f.SCB[y]) {
			return true
		}
	}
	return false
}

// Render turns a Frame into the RGB image an Apple IIgs would display, at
// the smallest canvas faithful to the data: 320×200 when every scanline is
// in 320 mode, 640×400 as soon as any scanline uses 640 mode (640-mode
// pixels need the full horizontal resolution, and doubling the scanlines
// keeps pixel shape consistent between the two widths).
//
// Rendering is the correctness anchor for the whole project: it honors
// per-line mode, per-line palette, the 640-mode pixel-position→palette-group
// mapping, and 320-mode color fill. No aspect-ratio correction is applied —
// SHR pixels are not square (see docs/SPEC.md); scaling for display is the
// caller's job.
func Render(f *Frame) *image.RGBA {
	if f.Uses640() {
		return Render640(f)
	}
	img, _ := Render320(f) // Uses640 already ruled out the only error
	return img
}

// Render320 renders onto a 320×200 canvas, one canvas pixel per SHR pixel.
// It fails if any scanline's SCB selects 640 mode: those lines carry 640
// horizontal positions and cannot be drawn at half resolution.
func Render320(f *Frame) (*image.RGBA, error) {
	if f.Uses640() {
		return nil, errors.New("frame contains 640-mode scanlines")
	}
	img := image.NewRGBA(image.Rect(0, 0, Width320, Height))
	for y := 0; y < Height; y++ {
		row := line320Colors(f, y)
		for x, c := range row {
			setPixel(img, x, y, c)
		}
	}
	return img, nil
}

// Render640 renders onto a 640×400 canvas: every scanline is drawn twice
// (canvas rows 2y and 2y+1), 640-mode pixels map one per canvas pixel, and
// 320-mode pixels are doubled horizontally. This is the only canvas that can
// represent frames mixing both modes, and it doubles a pure 320-mode frame
// in both dimensions.
func Render640(f *Frame) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, Width640, 2*Height))
	for y := 0; y < Height; y++ {
		scb := f.SCB[y]
		if SCBIs640(scb) {
			pal := &f.Palettes[SCBPalette(scb)]
			vals := UnpackLine640(f.Line(y))
			for x, v := range vals {
				c := pal[PaletteIndex640(x, v)].Color()
				setPixel(img, x, 2*y, c)
				setPixel(img, x, 2*y+1, c)
			}
			continue
		}
		row := line320Colors(f, y)
		for x, c := range row {
			setPixel(img, 2*x, 2*y, c)
			setPixel(img, 2*x+1, 2*y, c)
			setPixel(img, 2*x, 2*y+1, c)
			setPixel(img, 2*x+1, 2*y+1, c)
		}
	}
	return img
}

// line320Colors resolves a 320-mode scanline to its 320 colors, honoring the
// line's palette and color-fill mode.
func line320Colors(f *Frame, y int) [Width320]color.RGBA {
	scb := f.SCB[y]
	pal := &f.Palettes[SCBPalette(scb)]
	idx := UnpackLine320(f.Line(y))
	fill := SCBIsFill(scb)

	var out [Width320]color.RGBA
	// Color fill mode: pixel value 0 repeats the previous pixel's color
	// instead of meaning "palette entry 0". Hardware behavior is undefined
	// when the leftmost pixel is 0; we render palette entry 0.
	prev := pal[0].Color()
	for x, v := range idx {
		c := pal[v].Color()
		if fill && v == 0 && x > 0 {
			c = prev
		}
		prev = c
		out[x] = c
	}
	return out
}

func setPixel(img *image.RGBA, x, y int, c color.RGBA) {
	off := img.PixOffset(x, y)
	img.Pix[off] = c.R
	img.Pix[off+1] = c.G
	img.Pix[off+2] = c.B
	img.Pix[off+3] = c.A
}
