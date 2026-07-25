package shr

import "fmt"

// group640 maps a pixel's position within its byte (x mod 4) to the base of
// the group of four palette entries that pixel can reach in 640 mode:
//
//	pixel position in byte:  0     1     2     3
//	palette entries usable:  8-11  12-15 0-3   4-7
//
// This is the single most error-prone fact about 640 mode: the 2-bit pixel
// value does NOT index the palette directly.
var group640 = [4]uint8{8, 12, 0, 4}

// PaletteIndex640 resolves a 640-mode pixel to its palette entry. x is the
// absolute pixel x coordinate on the scanline (0–639); value is the 2-bit
// pixel value. Every 640-mode reader and writer in the program must go
// through this function.
func PaletteIndex640(x int, value uint8) uint8 {
	return group640[x&3] + value&3
}

// PackLine320 packs 320 4-bit palette indices into 160 bytes. One byte holds
// two pixels; the HIGH nibble is the LEFT pixel. len(idx) must be 320 and
// len(dst) at least 160.
func PackLine320(dst []byte, idx []uint8) error {
	if len(idx) != Width320 {
		return fmt.Errorf("PackLine320: need %d pixel values, got %d", Width320, len(idx))
	}
	if len(dst) < BytesPerLine {
		return fmt.Errorf("PackLine320: dst too short: %d < %d", len(dst), BytesPerLine)
	}
	for i := 0; i < BytesPerLine; i++ {
		dst[i] = idx[2*i]&0x0F<<4 | idx[2*i+1]&0x0F
	}
	return nil
}

// PackLine640 packs 640 2-bit pixel values into 160 bytes. One byte holds
// four pixels; bits 7–6 are the LEFTMOST pixel. The values are raw 2-bit
// pixel values, not palette indices — the position-dependent palette mapping
// (PaletteIndex640) is applied by whoever chose the values and by whoever
// reads them back. len(val) must be 640 and len(dst) at least 160.
func PackLine640(dst []byte, val []uint8) error {
	if len(val) != Width640 {
		return fmt.Errorf("PackLine640: need %d pixel values, got %d", Width640, len(val))
	}
	if len(dst) < BytesPerLine {
		return fmt.Errorf("PackLine640: dst too short: %d < %d", len(dst), BytesPerLine)
	}
	for i := 0; i < BytesPerLine; i++ {
		dst[i] = val[4*i]&3<<6 | val[4*i+1]&3<<4 | val[4*i+2]&3<<2 | val[4*i+3]&3
	}
	return nil
}

// UnpackLine320 unpacks 160 bytes into 320 4-bit palette indices
// (high nibble first — the left pixel).
func UnpackLine320(src []byte) [Width320]uint8 {
	var out [Width320]uint8
	for i := 0; i < BytesPerLine; i++ {
		out[2*i] = src[i] >> 4
		out[2*i+1] = src[i] & 0x0F
	}
	return out
}

// UnpackLine640 unpacks 160 bytes into 640 raw 2-bit pixel values
// (bits 7–6 first — the leftmost pixel). Resolve them to palette entries
// with PaletteIndex640.
func UnpackLine640(src []byte) [Width640]uint8 {
	var out [Width640]uint8
	for i := 0; i < BytesPerLine; i++ {
		b := src[i]
		out[4*i] = b >> 6
		out[4*i+1] = b >> 4 & 3
		out[4*i+2] = b >> 2 & 3
		out[4*i+3] = b & 3
	}
	return out
}
