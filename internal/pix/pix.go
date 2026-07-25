// Package pix provides the linear-light float image buffer the conversion
// pipeline works in. Quantization and error diffusion behave correctly in
// linear light, not in gamma-encoded 8-bit sRGB, so decoding converts once
// on the way in and everything downstream stays linear until final palette
// matching.
package pix

import (
	"image"
	"math"
)

// LinearImage is a linear-light RGB image, 3 float32 channels per pixel in
// row-major RGB order. Channel values are nominally 0.0–1.0.
type LinearImage struct {
	W, H int
	Pix  []float32 // len = 3*W*H
}

// New returns a black (all-zero) linear image.
func New(w, h int) *LinearImage {
	return &LinearImage{W: w, H: h, Pix: make([]float32, 3*w*h)}
}

// At returns the linear RGB value at (x, y).
func (p *LinearImage) At(x, y int) (r, g, b float32) {
	i := 3 * (y*p.W + x)
	return p.Pix[i], p.Pix[i+1], p.Pix[i+2]
}

// Set stores the linear RGB value at (x, y).
func (p *LinearImage) Set(x, y int, r, g, b float32) {
	i := 3 * (y*p.W + x)
	p.Pix[i], p.Pix[i+1], p.Pix[i+2] = r, g, b
}

// FromImage converts a decoded image to linear light. Alpha is composited
// over black (image.Image's RGBA() is alpha-premultiplied, which is exactly
// over-black compositing when alpha is then dropped).
func FromImage(img image.Image) *LinearImage {
	b := img.Bounds()
	out := New(b.Dx(), b.Dy())
	i := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA() // 16-bit premultiplied
			out.Pix[i] = SRGBToLinear(float32(r) / 65535)
			out.Pix[i+1] = SRGBToLinear(float32(g) / 65535)
			out.Pix[i+2] = SRGBToLinear(float32(bl) / 65535)
			i += 3
		}
	}
	return out
}

// SRGBToLinear applies the sRGB electro-optical transfer function.
func SRGBToLinear(v float32) float32 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return float32(math.Pow((float64(v)+0.055)/1.055, 2.4))
}

// LinearToSRGB applies the inverse sRGB transfer function.
func LinearToSRGB(v float32) float32 {
	if v <= 0.0031308 {
		return v * 12.92
	}
	return float32(1.055*math.Pow(float64(v), 1/2.4) - 0.055)
}

// Clamp01 clamps v to [0, 1].
func Clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
