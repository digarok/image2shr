package bmp

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"testing"
)

// makeBMP assembles a BMP file from parts: a BITMAPFILEHEADER, a 40-byte
// BITMAPINFOHEADER, an optional palette (raw B,G,R,X quads), and raw
// (pre-padded) pixel rows in on-disk order.
func makeBMP(width, height int32, bpp uint16, clrUsed uint32, palette, rows []byte) []byte {
	var buf bytes.Buffer
	dataOffset := uint32(14 + 40 + len(palette))

	// BITMAPFILEHEADER
	buf.WriteString("BM")
	le := binary.LittleEndian
	b4 := func(v uint32) { var t [4]byte; le.PutUint32(t[:], v); buf.Write(t[:]) }
	b2 := func(v uint16) { var t [2]byte; le.PutUint16(t[:], v); buf.Write(t[:]) }
	b4(dataOffset + uint32(len(rows))) // file size
	b4(0)                              // reserved
	b4(dataOffset)

	// BITMAPINFOHEADER
	b4(40)
	b4(uint32(width))
	b4(uint32(height))
	b2(1) // planes
	b2(bpp)
	b4(0) // compression = BI_RGB
	b4(uint32(len(rows)))
	b4(2835) // x pixels/meter
	b4(2835) // y pixels/meter
	b4(clrUsed)
	b4(0) // colors important

	buf.Write(palette)
	buf.Write(rows)
	return buf.Bytes()
}

var (
	red   = color.RGBA{0xFF, 0, 0, 0xFF}
	green = color.RGBA{0, 0xFF, 0, 0xFF}
	blue  = color.RGBA{0, 0, 0xFF, 0xFF}
	white = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	black = color.RGBA{0, 0, 0, 0xFF}
)

func checkPixels(t *testing.T, data []byte, w, h int, want [][]color.RGBA) {
	t.Helper()
	img, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != w || img.Bounds().Dy() != h {
		t.Fatalf("bounds = %v, want %dx%d", img.Bounds(), w, h)
	}
	for y, rowWant := range want {
		for x, c := range rowWant {
			got := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			if got != c {
				t.Errorf("pixel (%d,%d) = %v, want %v", x, y, got, c)
			}
		}
	}
}

func TestDecode24BitBottomUp(t *testing.T) {
	// 3×2, stride = 12 (9 pixel bytes + 3 pad). Rows on disk bottom-first,
	// pixels stored B,G,R.
	rows := []byte{
		// disk row 0 = image row y=1: white, black, red
		0xFF, 0xFF, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xAA, 0xBB, 0xCC, // pad junk
		// disk row 1 = image row y=0: red, green, blue
		0x00, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0xFF, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	data := makeBMP(3, 2, 24, 0, nil, rows)
	checkPixels(t, data, 3, 2, [][]color.RGBA{
		{red, green, blue},
		{white, black, red},
	})
}

func TestDecode32BitTopDown(t *testing.T) {
	// 2×2, height = -2 (top-down), stride = 8, pixels B,G,R,X (X ignored).
	rows := []byte{
		0x00, 0x00, 0xFF, 0x7F, 0x00, 0xFF, 0x00, 0x00, // image row 0: red, green
		0xFF, 0x00, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0x00, // image row 1: blue, white
	}
	data := makeBMP(2, -2, 32, 0, nil, rows)
	checkPixels(t, data, 2, 2, [][]color.RGBA{
		{red, green},
		{blue, white},
	})
}

func TestDecode8BitPaletted(t *testing.T) {
	// 3×2 bottom-up, 4-entry palette (clrUsed=4), stride = 4 (3 + 1 pad).
	palette := []byte{
		0x00, 0x00, 0x00, 0x00, // 0: black (B,G,R,X)
		0x00, 0x00, 0xFF, 0x00, // 1: red
		0x00, 0xFF, 0x00, 0x00, // 2: green
		0xFF, 0x00, 0x00, 0x00, // 3: blue
	}
	rows := []byte{
		3, 0, 1, 0xEE, // disk row 0 = image row y=1: blue, black, red
		0, 1, 2, 0x00, // disk row 1 = image row y=0: black, red, green
	}
	data := makeBMP(3, 2, 8, 4, palette, rows)
	checkPixels(t, data, 3, 2, [][]color.RGBA{
		{black, red, green},
		{blue, black, red},
	})
}

func TestDecode8BitFullPalette(t *testing.T) {
	// clrUsed = 0 means a full 256-entry palette.
	palette := make([]byte, 256*4)
	palette[42*4+2] = 0xFF // entry 42 = red
	rows := []byte{42, 0, 0, 0}
	data := makeBMP(1, 1, 8, 0, palette, rows)
	checkPixels(t, data, 1, 1, [][]color.RGBA{{red}})
}

func TestDecodeErrors(t *testing.T) {
	good := makeBMP(2, 2, 24, 0, nil, make([]byte, 16))
	tests := []struct {
		name   string
		mangle func([]byte) []byte
	}{
		{"bad magic", func(d []byte) []byte { d[0] = 'X'; return d }},
		{"RLE8 compression", func(d []byte) []byte { d[14+16] = 1; return d }},
		{"16 bpp", func(d []byte) []byte { d[14+14] = 16; return d }},
		{"core header (size 12)", func(d []byte) []byte { d[14] = 12; return d }},
		{"zero height", func(d []byte) []byte {
			binary.LittleEndian.PutUint32(d[14+8:], 0)
			return d
		}},
		{"negative width", func(d []byte) []byte {
			binary.LittleEndian.PutUint32(d[14+4:], 0xFFFFFFFE) // -2
			return d
		}},
		{"truncated pixel data", func(d []byte) []byte { return d[:len(d)-8] }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := tt.mangle(append([]byte(nil), good...))
			if _, err := Decode(bytes.NewReader(data)); err == nil {
				t.Error("expected decode error, got nil")
			}
		})
	}
}

func TestDecodePaletteIndexOutOfRange(t *testing.T) {
	palette := []byte{0, 0, 0, 0, 0, 0, 0xFF, 0} // 2 entries
	rows := []byte{5, 0, 0, 0}                   // index 5 > palette size
	data := makeBMP(1, 1, 8, 2, palette, rows)
	if _, err := Decode(bytes.NewReader(data)); err == nil {
		t.Error("expected error for out-of-range palette index")
	}
}

func TestDecodeConfig(t *testing.T) {
	data := makeBMP(7, -3, 32, 0, nil, make([]byte, 7*4*3))
	cfg, err := DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 7 || cfg.Height != 3 {
		t.Errorf("config = %dx%d, want 7x3", cfg.Width, cfg.Height)
	}
}

// TestDecodeSkipsHeaderGap checks that a nonstandard gap between the palette
// and the declared pixel offset is skipped correctly.
func TestDecodeSkipsHeaderGap(t *testing.T) {
	rows := []byte{0x00, 0x00, 0xFF, 0x00} // one red pixel, 1 pad byte... (1*3+31)/32*4 = 4
	data := makeBMP(1, 1, 24, 0, nil, rows)
	// Insert an 8-byte gap before pixel data and bump the declared offset.
	offset := binary.LittleEndian.Uint32(data[10:14])
	withGap := append(append(append([]byte(nil), data[:offset]...), make([]byte, 8)...), data[offset:]...)
	binary.LittleEndian.PutUint32(withGap[10:14], offset+8)
	checkPixels(t, withGap, 1, 1, [][]color.RGBA{{red}})
}
