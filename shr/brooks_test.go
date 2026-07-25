package shr

import (
	"bytes"
	"testing"
)

// TestBrooksReversedPaletteOrder pins the format's defining quirk: each
// line's 16 colors are stored highest entry first.
func TestBrooksReversedPaletteOrder(t *testing.T) {
	f := &Frame{LinePalettes: &[Height][16]RGB12{}}
	f.LinePalettes[0][0] = RGB12{R: 15, G: 8, B: 0}  // $0F80 → bytes $80 $0F
	f.LinePalettes[0][15] = RGB12{R: 1, G: 2, B: 3}  // $0123 → bytes $23 $01
	f.LinePalettes[199][7] = RGB12{R: 4, G: 5, B: 6} // $0456 → bytes $56 $04

	buf, err := f.EncodeBrooks()
	if err != nil {
		t.Fatal(err)
	}
	if len(buf) != BrooksSize {
		t.Fatalf("encoded %d bytes, want %d", len(buf), BrooksSize)
	}
	pal := PixelDataSize
	// Line 0: entry 15 occupies the first pair, entry 0 the last.
	if buf[pal] != 0x23 || buf[pal+1] != 0x01 {
		t.Errorf("line 0 entry 15 stored as % X, want 23 01 at start of palette", buf[pal:pal+2])
	}
	if buf[pal+30] != 0x80 || buf[pal+31] != 0x0F {
		t.Errorf("line 0 entry 0 stored as % X, want 80 0F at end of palette", buf[pal+30:pal+32])
	}
	// Line 199, entry 7 → offset pal + 199*32 + (15-7)*2.
	off := pal + 199*32 + 8*2
	if buf[off] != 0x56 || buf[off+1] != 0x04 {
		t.Errorf("line 199 entry 7 stored as % X, want 56 04", buf[off:off+2])
	}
}

func TestBrooksRoundTrip(t *testing.T) {
	f := &Frame{LinePalettes: &[Height][16]RGB12{}}
	for y := 0; y < Height; y++ {
		for c := 0; c < 16; c++ {
			f.LinePalettes[y][c] = RGB12{R: uint8(y % 16), G: uint8(c), B: uint8((y + c) % 16)}
		}
		f.Pixels[y*BytesPerLine] = byte(y)
	}
	buf, err := f.EncodeBrooks()
	if err != nil {
		t.Fatal(err)
	}
	g, err := DecodeBrooks(buf)
	if err != nil {
		t.Fatal(err)
	}
	if *g.LinePalettes != *f.LinePalettes {
		t.Error("palettes did not survive the round trip")
	}
	if g.Pixels != f.Pixels {
		t.Error("pixels did not survive the round trip")
	}
	buf2, err := g.EncodeBrooks()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf, buf2) {
		t.Error("re-encoding is not byte-identical")
	}
}

// TestBrooksExpandsSCBFrame: a normal 16-palette frame encodes by expanding
// each line's SCB-selected palette into that line's slot.
func TestBrooksExpandsSCBFrame(t *testing.T) {
	f := &Frame{}
	f.Palettes[3][5] = RGB12{R: 9, G: 9, B: 9}
	f.SCB[42] = 0x03 // line 42 uses palette 3

	buf, err := f.EncodeBrooks()
	if err != nil {
		t.Fatal(err)
	}
	g, err := DecodeBrooks(buf)
	if err != nil {
		t.Fatal(err)
	}
	want := RGB12{R: 9, G: 9, B: 9}
	if g.LinePalettes[42][5] != want {
		t.Errorf("line 42 entry 5 = %v, want %v (expanded from palette 3)", g.LinePalettes[42][5], want)
	}
	if g.LinePalettes[0][5] != (RGB12{}) {
		t.Errorf("line 0 entry 5 = %v, want zero (palette 0)", g.LinePalettes[0][5])
	}
}

func TestBrooksRejects640(t *testing.T) {
	f := &Frame{}
	f.SCB[10] = SCBMode640
	if _, err := f.EncodeBrooks(); err == nil {
		t.Error("EncodeBrooks accepted a 640-mode frame")
	}
}

func TestBrooksDecodeWrongSize(t *testing.T) {
	if _, err := DecodeBrooks(make([]byte, FrameSize)); err == nil {
		t.Error("DecodeBrooks accepted a 32768-byte input")
	}
}

// TestRenderUsesLinePalettes: the same pixel value must render through each
// line's own palette on a 3200-color frame.
func TestRenderUsesLinePalettes(t *testing.T) {
	f := &Frame{LinePalettes: &[Height][16]RGB12{}}
	f.LinePalettes[0][1] = RGB12{R: 15, G: 0, B: 0}
	f.LinePalettes[1][1] = RGB12{R: 0, G: 15, B: 0}
	f.Pixels[0] = 0x11            // line 0: pixels 0,1 = value 1
	f.Pixels[BytesPerLine] = 0x11 // line 1: same values
	img := Render(f)
	if b := img.Bounds(); b.Dx() != Width320 || b.Dy() != Height {
		t.Fatalf("canvas %dx%d, want %dx%d", b.Dx(), b.Dy(), Width320, Height)
	}
	r0, _, _, _ := img.At(0, 0).RGBA()
	_, g1, _, _ := img.At(0, 1).RGBA()
	if r0 != 0xFFFF {
		t.Errorf("line 0 should render red from its own palette (r=%04X)", r0)
	}
	if g1 != 0xFFFF {
		t.Errorf("line 1 should render green from its own palette (g=%04X)", g1)
	}
}
