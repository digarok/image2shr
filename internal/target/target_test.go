package target

import (
	"testing"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"

	_ "github.com/digarok/image2shr/internal/dither" // register ditherers
)

func TestGrey16Registered(t *testing.T) {
	tgt, err := pipeline.LookupTarget("shr320-grey16")
	if err != nil {
		t.Fatal(err)
	}
	if w, h := tgt.Geometry(); w != 320 || h != 200 {
		t.Errorf("geometry = %dx%d, want 320x200", w, h)
	}
}

func TestColor256Registered(t *testing.T) {
	tgt, err := pipeline.LookupTarget("shr320-color256")
	if err != nil {
		t.Fatal(err)
	}
	if w, h := tgt.Geometry(); w != 320 || h != 200 {
		t.Errorf("geometry = %dx%d, want 320x200", w, h)
	}
}

func TestGrey16EndToEnd(t *testing.T) {
	tgt, _ := pipeline.LookupTarget("shr320-grey16")
	src := pix.New(320, 200)
	// Left half black, right half white.
	for y := 0; y < 200; y++ {
		for x := 160; x < 320; x++ {
			src.Set(x, y, 1, 1, 1)
		}
	}
	f, err := tgt.Convert(src, pipeline.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}

	// Milestone spec: all SCBs $00, palette 0 = $0000…$0FFF ramp.
	for y, scb := range f.SCB {
		if scb != 0x00 {
			t.Fatalf("SCB[%d] = $%02X, want $00", y, scb)
		}
	}
	for n := 0; n < 16; n++ {
		want := shr.RGB12{R: uint8(n), G: uint8(n), B: uint8(n)}
		if f.Palettes[0][n] != want {
			t.Errorf("palette 0 entry %d = %v, want %v", n, f.Palettes[0][n], want)
		}
	}
	// Black half packs to $00 bytes, white half to $FF.
	if f.Pixels[0] != 0x00 {
		t.Errorf("left-edge byte = $%02X, want $00", f.Pixels[0])
	}
	if f.Pixels[159] != 0xFF {
		t.Errorf("right-edge byte = $%02X, want $FF", f.Pixels[159])
	}
}

func TestGrey16BadLuma(t *testing.T) {
	tgt, _ := pipeline.LookupTarget("shr320-grey16")
	opt := pipeline.DefaultOptions()
	opt.Luma = "bogus"
	if _, err := tgt.Convert(pix.New(320, 200), opt); err == nil {
		t.Error("expected error for unknown luma")
	}
}
