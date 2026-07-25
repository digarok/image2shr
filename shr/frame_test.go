package shr

import (
	"bytes"
	"testing"
)

func TestRGB12Bytes(t *testing.T) {
	tests := []struct {
		name string
		c    RGB12
		want [2]byte
	}{
		// The spec's worked example: $0F80 is stored as bytes $80 $0F.
		{"spec example $0F80", RGB12{R: 0xF, G: 0x8, B: 0x0}, [2]byte{0x80, 0x0F}},
		{"black", RGB12{0, 0, 0}, [2]byte{0x00, 0x00}},
		{"white", RGB12{15, 15, 15}, [2]byte{0xFF, 0x0F}},
		{"pure blue", RGB12{0, 0, 15}, [2]byte{0x0F, 0x00}},
		{"pure green", RGB12{0, 15, 0}, [2]byte{0xF0, 0x00}},
		{"pure red", RGB12{15, 0, 0}, [2]byte{0x00, 0x0F}},
		{"mixed $0123", RGB12{1, 2, 3}, [2]byte{0x23, 0x01}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.Bytes(); got != tt.want {
				t.Errorf("Bytes(%v) = %02X %02X, want %02X %02X",
					tt.c, got[0], got[1], tt.want[0], tt.want[1])
			}
			if back := RGB12FromBytes(tt.want[0], tt.want[1]); back != tt.c {
				t.Errorf("RGB12FromBytes(%02X, %02X) = %v, want %v",
					tt.want[0], tt.want[1], back, tt.c)
			}
		})
	}
}

func TestRGB12FromBytesIgnoresHighNibble(t *testing.T) {
	// High nibble of the high byte must be zero on hardware; tolerate junk.
	if got := RGB12FromBytes(0x23, 0xF1); got != (RGB12{1, 2, 3}) {
		t.Errorf("got %v, want {1 2 3}", got)
	}
}

func TestRGB12Color(t *testing.T) {
	c := RGB12{0xF, 0x8, 0x0}.Color()
	if c.R != 0xFF || c.G != 0x88 || c.B != 0x00 || c.A != 0xFF {
		t.Errorf("nibble doubling wrong: %v", c)
	}
}

func TestRGB12String(t *testing.T) {
	if s := (RGB12{0xF, 0x8, 0x0}).String(); s != "$0F80" {
		t.Errorf("String() = %q, want $0F80", s)
	}
}

func TestEncodeRawLayout(t *testing.T) {
	f := &Frame{}
	f.Pixels[0] = 0xAA
	f.Pixels[PixelDataSize-1] = 0xBB
	f.SCB[0] = 0x01
	f.SCB[199] = 0x0F
	f.Palettes[0][0] = RGB12{1, 2, 3}    // first palette entry
	f.Palettes[15][15] = RGB12{15, 8, 0} // last palette entry

	buf := f.EncodeRaw()
	if len(buf) != FrameSize {
		t.Fatalf("EncodeRaw length = %d, want %d", len(buf), FrameSize)
	}
	checks := []struct {
		off  int
		want byte
		what string
	}{
		{0x0000, 0xAA, "first pixel byte"},
		{0x7CFF, 0xBB, "last pixel byte"},
		{0x7D00, 0x01, "SCB line 0"},
		{0x7DC7, 0x0F, "SCB line 199"},
		{0x7E00, 0x23, "palette 0 color 0 low byte (GB)"},
		{0x7E01, 0x01, "palette 0 color 0 high byte (0R)"},
		{0x7FFE, 0x80, "palette 15 color 15 low byte"},
		{0x7FFF, 0x0F, "palette 15 color 15 high byte"},
	}
	for _, c := range checks {
		if buf[c.off] != c.want {
			t.Errorf("byte at $%04X (%s) = $%02X, want $%02X", c.off, c.what, buf[c.off], c.want)
		}
	}
	// Reserved area $7DC8–$7DFF must be all zero.
	if !bytes.Equal(buf[reservedOffset:paletteOffset], make([]byte, paletteOffset-reservedOffset)) {
		t.Error("reserved area $7DC8-$7DFF is not all zero")
	}
}

func TestRawRoundTrip(t *testing.T) {
	f := &Frame{}
	for i := range f.Pixels {
		f.Pixels[i] = byte(i * 7)
	}
	for y := range f.SCB {
		f.SCB[y] = MakeSCB(y%2 == 0, false, y%5 == 0, uint8(y%16))
	}
	for p := 0; p < 16; p++ {
		for c := 0; c < 16; c++ {
			f.Palettes[p][c] = RGB12{uint8(p), uint8(c), uint8((p + c) % 16)}
		}
	}
	got, err := DecodeRaw(f.EncodeRaw())
	if err != nil {
		t.Fatal(err)
	}
	if *got != *f {
		t.Error("DecodeRaw(EncodeRaw(f)) != f")
	}
}

func TestDecodeRawWrongSize(t *testing.T) {
	if _, err := DecodeRaw(make([]byte, 32767)); err == nil {
		t.Error("expected error for 32767-byte input")
	}
}
