package shr

import (
	"image/color"
	"testing"
)

func pixelAt(t *testing.T, img interface {
	At(x, y int) color.Color
}, x, y int) color.RGBA {
	t.Helper()
	r, g, b, a := img.At(x, y).RGBA()
	return color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func TestRender320(t *testing.T) {
	f := &Frame{}
	f.Palettes[0][0] = RGB12{0, 0, 0}
	f.Palettes[0][5] = RGB12{15, 0, 0}
	f.Palettes[3][5] = RGB12{0, 15, 0} // same index, different palette
	f.SCB[1] = MakeSCB(false, false, false, 3)

	// Line 0 (palette 0) and line 1 (palette 3): first pixel = index 5.
	f.Pixels[0] = 0x50 // line 0: left pixel 5, right pixel 0
	f.Pixels[BytesPerLine] = 0x50

	img := Render(f)
	if img.Bounds().Dx() != 640 || img.Bounds().Dy() != 200 {
		t.Fatalf("canvas = %v, want 640x200", img.Bounds())
	}
	red := color.RGBA{0xFF, 0, 0, 0xFF}
	green := color.RGBA{0, 0xFF, 0, 0xFF}
	black := color.RGBA{0, 0, 0, 0xFF}

	// 320-mode pixel 0 is doubled onto canvas x=0 and x=1.
	if got := pixelAt(t, img, 0, 0); got != red {
		t.Errorf("(0,0) = %v, want red", got)
	}
	if got := pixelAt(t, img, 1, 0); got != red {
		t.Errorf("(1,0) = %v, want red (320 pixels are doubled)", got)
	}
	if got := pixelAt(t, img, 2, 0); got != black {
		t.Errorf("(2,0) = %v, want black", got)
	}
	// Line 1 uses palette 3 via its SCB.
	if got := pixelAt(t, img, 0, 1); got != green {
		t.Errorf("(0,1) = %v, want green (per-line palette)", got)
	}
}

// TestRender640PositionMapping proves the renderer resolves 640-mode pixels
// through the position-dependent palette groups: the SAME 2-bit value at four
// consecutive x positions must produce four DIFFERENT palette entries.
func TestRender640PositionMapping(t *testing.T) {
	f := &Frame{}
	f.SCB[0] = MakeSCB(true, false, false, 0)
	// Distinct colors in each group's entry for value 1: 9, 13, 1, 5.
	f.Palettes[0][9] = RGB12{15, 0, 0}  // position 0 → entries 8-11
	f.Palettes[0][13] = RGB12{0, 15, 0} // position 1 → entries 12-15
	f.Palettes[0][1] = RGB12{0, 0, 15}  // position 2 → entries 0-3
	f.Palettes[0][5] = RGB12{15, 15, 0} // position 3 → entries 4-7
	// One byte with all four pixels = value 1: 01 01 01 01 = $55.
	f.Pixels[0] = 0x55

	img := Render(f)
	want := []color.RGBA{
		{0xFF, 0, 0, 0xFF},
		{0, 0xFF, 0, 0xFF},
		{0, 0, 0xFF, 0xFF},
		{0xFF, 0xFF, 0, 0xFF},
	}
	for x, w := range want {
		if got := pixelAt(t, img, x, 0); got != w {
			t.Errorf("640 pixel x=%d = %v, want %v (position→group mapping)", x, got, w)
		}
	}
}

func TestRenderFillMode(t *testing.T) {
	f := &Frame{}
	f.Palettes[0][0] = RGB12{0, 0, 15}         // entry 0 is blue
	f.Palettes[0][7] = RGB12{15, 0, 0}         // entry 7 is red
	f.SCB[0] = MakeSCB(false, false, true, 0)  // fill on
	f.SCB[1] = MakeSCB(false, false, false, 0) // fill off, same pixels
	line := []byte{0x70, 0x00}                 // pixels: 7, 0, 0, 0, then zeros
	copy(f.Pixels[0:], line)
	copy(f.Pixels[BytesPerLine:], line)

	img := Render(f)
	red := color.RGBA{0xFF, 0, 0, 0xFF}
	blue := color.RGBA{0, 0, 0xFF, 0xFF}

	// Fill line: value 0 after the red pixel repeats red.
	if got := pixelAt(t, img, 2, 0); got != red {
		t.Errorf("fill line pixel 1 = %v, want red (0 repeats previous color)", got)
	}
	if got := pixelAt(t, img, 6, 0); got != red {
		t.Errorf("fill line pixel 3 = %v, want red (fill keeps repeating)", got)
	}
	// Non-fill line: value 0 is palette entry 0 (blue).
	if got := pixelAt(t, img, 2, 1); got != blue {
		t.Errorf("non-fill line pixel 1 = %v, want blue (palette entry 0)", got)
	}
}
