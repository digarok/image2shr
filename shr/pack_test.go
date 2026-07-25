package shr

import "testing"

// TestPaletteIndex640 pins the position-dependent palette mapping — the most
// error-prone fact about 640 mode. From the spec:
//
//	pixel position in byte:  0     1     2     3
//	palette entries usable:  8-11  12-15 0-3   4-7
func TestPaletteIndex640(t *testing.T) {
	base := [4]uint8{8, 12, 0, 4}
	for pos := 0; pos < 4; pos++ {
		for v := uint8(0); v < 4; v++ {
			want := base[pos] + v
			// Check several absolute x sharing this position within a byte.
			for _, x := range []int{pos, 4 + pos, 636 + pos} {
				if got := PaletteIndex640(x, v); got != want {
					t.Errorf("PaletteIndex640(x=%d, v=%d) = %d, want %d", x, v, got, want)
				}
			}
		}
	}
}

func TestPackLine320(t *testing.T) {
	idx := make([]uint8, Width320)
	idx[0], idx[1] = 0xA, 0xB // first byte: high nibble = left pixel
	idx[318], idx[319] = 0x1, 0x2
	dst := make([]byte, BytesPerLine)
	if err := PackLine320(dst, idx); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 0xAB {
		t.Errorf("byte 0 = $%02X, want $AB (high nibble is the LEFT pixel)", dst[0])
	}
	if dst[159] != 0x12 {
		t.Errorf("byte 159 = $%02X, want $12", dst[159])
	}
	got := UnpackLine320(dst)
	for i, v := range idx {
		if got[i] != v {
			t.Fatalf("round trip mismatch at pixel %d: got %d want %d", i, got[i], v)
		}
	}
}

func TestPackLine640(t *testing.T) {
	val := make([]uint8, Width640)
	val[0], val[1], val[2], val[3] = 3, 2, 1, 0 // bits 7-6 = leftmost pixel
	dst := make([]byte, BytesPerLine)
	if err := PackLine640(dst, val); err != nil {
		t.Fatal(err)
	}
	if dst[0] != 0xE4 { // 11 10 01 00
		t.Errorf("byte 0 = $%02X, want $E4 (bits 7-6 are the LEFTMOST pixel)", dst[0])
	}
	got := UnpackLine640(dst)
	for i, v := range val {
		if got[i] != v {
			t.Fatalf("round trip mismatch at pixel %d: got %d want %d", i, got[i], v)
		}
	}
}

func TestPackLineErrors(t *testing.T) {
	if err := PackLine320(make([]byte, BytesPerLine), make([]uint8, 100)); err == nil {
		t.Error("PackLine320: expected error for wrong pixel count")
	}
	if err := PackLine320(make([]byte, 10), make([]uint8, Width320)); err == nil {
		t.Error("PackLine320: expected error for short dst")
	}
	if err := PackLine640(make([]byte, BytesPerLine), make([]uint8, 100)); err == nil {
		t.Error("PackLine640: expected error for wrong pixel count")
	}
	if err := PackLine640(make([]byte, 10), make([]uint8, Width640)); err == nil {
		t.Error("PackLine640: expected error for short dst")
	}
}

func TestMakeSCB(t *testing.T) {
	tests := []struct {
		mode640, irq, fill bool
		pal                uint8
		want               byte
	}{
		{false, false, false, 0, 0x00},
		{true, false, false, 0, 0x80},
		{false, true, false, 0, 0x40},
		{false, false, true, 0, 0x20},
		{false, false, false, 15, 0x0F},
		{true, true, true, 5, 0xE5},
		{false, false, false, 0xFF, 0x0F}, // palette masked to 0-15
	}
	for _, tt := range tests {
		got := MakeSCB(tt.mode640, tt.irq, tt.fill, tt.pal)
		if got != tt.want {
			t.Errorf("MakeSCB(%v,%v,%v,%d) = $%02X, want $%02X",
				tt.mode640, tt.irq, tt.fill, tt.pal, got, tt.want)
		}
		if SCBIs640(got) != tt.mode640 || SCBIsInterrupt(got) != tt.irq ||
			SCBIsFill(got) != tt.fill || SCBPalette(got) != tt.pal&0x0F {
			t.Errorf("SCB accessors disagree with MakeSCB for $%02X", got)
		}
	}
}
