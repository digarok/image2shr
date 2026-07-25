package shr

import "fmt"

// Brooks 3200-color format (".3200", ProDOS type PIC $C1, auxtype $0002).
//
// 3200-color mode is a software trick, not a hardware mode: a display
// routine reloads the color tables while racing the beam, giving each of
// the 200 scanlines its own 16-color palette — 3200 simultaneous colors.
// The file is a fixed 38,400-byte layout:
//
//	$0000–$7CFF  32000 bytes  pixel data, 200 lines × 160 bytes (as raw SHR)
//	$7D00–$95FF   6400 bytes  200 palettes, one 32-byte palette per line,
//	                          colors stored in REVERSE order: entry 15
//	                          first, entry 0 last (each color is the usual
//	                          little-endian $0RGB pair)
//
// There is no SCB table: every line implicitly uses its own palette, in
// 320 mode, without fill or interrupts.
const (
	BrooksSize          = PixelDataSize + Height*32 // 38400 bytes total
	brooksPaletteOffset = PixelDataSize             // palettes follow the pixels
)

// brooksEntryOffset is where palette entry c of line y lives in the file:
// each line's 16 colors are stored highest entry first.
func brooksEntryOffset(y, c int) int {
	return brooksPaletteOffset + y*32 + (15-c)*2
}

// EncodeBrooks serializes the frame as a Brooks 3200-color file. A normal
// 16-palette frame is accepted too: each line's SCB-selected palette is
// expanded into that line's slot. Frames containing 640-mode scanlines
// cannot be expressed (Brooks is 320-mode only).
func (f *Frame) EncodeBrooks() ([]byte, error) {
	if f.Uses640() {
		return nil, fmt.Errorf("frame contains 640-mode scanlines; Brooks 3200 is 320-mode only")
	}
	buf := make([]byte, BrooksSize)
	copy(buf, f.Pixels[:])
	for y := 0; y < Height; y++ {
		pal := f.LinePalette(y)
		for c := 0; c < 16; c++ {
			b := pal[c].Bytes()
			off := brooksEntryOffset(y, c)
			buf[off] = b[0]
			buf[off+1] = b[1]
		}
	}
	return buf, nil
}

// DecodeBrooks parses a 38,400-byte Brooks 3200-color file into a Frame
// with LinePalettes set. SCBs and the 16-palette table stay zero — the
// format has neither.
func DecodeBrooks(data []byte) (*Frame, error) {
	if len(data) != BrooksSize {
		return nil, fmt.Errorf("Brooks 3200 file must be exactly %d bytes, got %d", BrooksSize, len(data))
	}
	f := &Frame{LinePalettes: &[Height][16]RGB12{}}
	copy(f.Pixels[:], data[:PixelDataSize])
	for y := 0; y < Height; y++ {
		for c := 0; c < 16; c++ {
			off := brooksEntryOffset(y, c)
			f.LinePalettes[y][c] = RGB12FromBytes(data[off], data[off+1])
		}
	}
	return f, nil
}
