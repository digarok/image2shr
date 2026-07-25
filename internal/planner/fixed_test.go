package planner

import (
	"testing"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

func TestGrey16Palette(t *testing.T) {
	pal := Grey16Palette()
	for n := 0; n < 16; n++ {
		want := shr.RGB12{R: uint8(n), G: uint8(n), B: uint8(n)}
		if pal[n] != want {
			t.Errorf("entry %d = %v, want %v", n, pal[n], want)
		}
	}
	// Spot-check the serialized form of entry 15: $0FFF → bytes $FF $0F.
	if b := pal[15].Bytes(); b != [2]byte{0xFF, 0x0F} {
		t.Errorf("entry 15 bytes = %02X %02X, want FF 0F", b[0], b[1])
	}
}

func TestFixedPlan(t *testing.T) {
	p := NewFixed("test", Grey16Palette())
	plan, err := p.Plan(pix.New(320, 200), pipeline.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode640 || plan.Fill {
		t.Error("fixed planner must produce 320 mode, fill off")
	}
	for y, pn := range plan.Line {
		if pn != 0 {
			t.Fatalf("line %d assigned palette %d, want 0", y, pn)
		}
	}
	if plan.Palettes[0] != Grey16Palette() {
		t.Error("palette 0 is not the grey ramp")
	}
	if plan.Palettes[1] != [16]shr.RGB12{} {
		t.Error("unused palettes should stay zero")
	}
}
