// Package shr models the Apple IIgs Super Hi-Res (SHR) screen: the Frame
// data structure, the 12-bit color format, scanline control bytes, pixel
// packers for both 320 and 640 modes, and a reference renderer.
//
// The SHR screen is a fixed 32,768-byte structure — a raw dump of IIgs RAM
// bank $E1, $2000–$9FFF. File offsets in an uncompressed .shr file
// (ProDOS type PIC $C1, auxtype $0000):
//
//	$0000–$7CFF  32000 bytes  pixel data, 200 lines × 160 bytes
//	$7D00–$7DC7    200 bytes  scanline control bytes (SCB), one per line
//	$7DC8–$7DFF     56 bytes  reserved, must be zero
//	$7E00–$7FFF    512 bytes  16 palettes × 16 colors × 2 bytes
package shr

import (
	"fmt"
	"image/color"
)

// Screen geometry and file layout constants.
const (
	Height       = 200 // scanlines
	Width320     = 320 // pixels per line in 320 mode (2 per byte)
	Width640     = 640 // pixels per line in 640 mode (4 per byte)
	BytesPerLine = 160 // pixel bytes per scanline, both modes

	PixelDataSize = Height * BytesPerLine // 32000 ($0000–$7CFF)
	FrameSize     = 32768                 // total uncompressed screen dump

	scbOffset      = 0x7D00 // 200 SCBs, one per line
	reservedOffset = 0x7DC8 // 56 reserved bytes, must be zero
	paletteOffset  = 0x7E00 // 16 palettes × 16 colors × 2 bytes
)

// RGB12 is one SHR palette color: 12-bit RGB444, each channel 0–15.
//
// On disk / in RAM a color is 16 bits, conceptually $0RGB, stored
// little-endian:
//
//	low byte  = GB  (high nibble green, low nibble blue)
//	high byte = 0R  (high nibble must be zero, low nibble red)
//
// So $0F80 (R=15 G=8 B=0) is stored as bytes $80 $0F.
type RGB12 struct{ R, G, B uint8 }

// Bytes returns the 2-byte little-endian $0RGB encoding: {GB, 0R}.
func (c RGB12) Bytes() [2]byte {
	return [2]byte{c.G&0x0F<<4 | c.B&0x0F, c.R & 0x0F}
}

// RGB12FromBytes decodes the little-endian $0RGB pair (lo=GB, hi=0R).
// The high nibble of hi is ignored (it must be zero on real hardware).
func RGB12FromBytes(lo, hi byte) RGB12 {
	return RGB12{R: hi & 0x0F, G: lo >> 4, B: lo & 0x0F}
}

// Color expands the 4-bit channels to 8 bits by nibble doubling
// ($N → $NN), the conventional way to display IIgs colors: 0 → 0x00,
// 15 → 0xFF, evenly spaced.
func (c RGB12) Color() color.RGBA {
	return color.RGBA{R: c.R * 0x11, G: c.G * 0x11, B: c.B * 0x11, A: 0xFF}
}

// String formats the color as $0RGB, e.g. "$0F80".
func (c RGB12) String() string {
	return fmt.Sprintf("$0%X%X%X", c.R&0x0F, c.G&0x0F, c.B&0x0F)
}

// Frame is the central data structure of the whole program: one complete
// SHR screen. Conversion produces a Frame, writers serialize a Frame, the
// renderer turns a Frame back into an RGB image.
type Frame struct {
	Pixels   [PixelDataSize]byte // raw packed pixel bytes, 200 lines × 160
	SCB      [Height]byte
	Palettes [16][16]RGB12
}

// Line returns the 160 packed pixel bytes of scanline y.
func (f *Frame) Line(y int) []byte {
	return f.Pixels[y*BytesPerLine : (y+1)*BytesPerLine]
}

// LinePalette returns the palette selected by scanline y's SCB.
func (f *Frame) LinePalette(y int) *[16]RGB12 {
	return &f.Palettes[f.SCB[y]&SCBPaletteMask]
}

// EncodeRaw serializes the frame into the 32,768-byte uncompressed screen
// dump laid out as documented at the top of this file.
func (f *Frame) EncodeRaw() []byte {
	buf := make([]byte, FrameSize)
	copy(buf, f.Pixels[:])
	copy(buf[scbOffset:], f.SCB[:])
	// buf[reservedOffset:paletteOffset] stays zero (reserved area).
	for p := 0; p < 16; p++ {
		for c := 0; c < 16; c++ {
			b := f.Palettes[p][c].Bytes()
			off := paletteOffset + p*32 + c*2
			buf[off] = b[0]
			buf[off+1] = b[1]
		}
	}
	return buf
}

// DecodeRaw parses a 32,768-byte uncompressed screen dump into a Frame.
// Nonzero bytes in the reserved area are tolerated (some tools leave junk
// there); they are not preserved.
func DecodeRaw(data []byte) (*Frame, error) {
	if len(data) != FrameSize {
		return nil, fmt.Errorf("raw SHR screen must be exactly %d bytes, got %d", FrameSize, len(data))
	}
	f := &Frame{}
	copy(f.Pixels[:], data[:PixelDataSize])
	copy(f.SCB[:], data[scbOffset:scbOffset+Height])
	for p := 0; p < 16; p++ {
		for c := 0; c < 16; c++ {
			off := paletteOffset + p*32 + c*2
			f.Palettes[p][c] = RGB12FromBytes(data[off], data[off+1])
		}
	}
	return f, nil
}
