package pipeline

import (
	"testing"

	"github.com/digarok/image2shr/shr"
)

func TestPackFrame320(t *testing.T) {
	var plan Plan
	plan.Palettes[2][5] = shr.RGB12{R: 1, G: 2, B: 3}
	for y := range plan.Line {
		plan.Line[y] = 2
	}
	idx := Indexed{W: 320, H: 200, Pix: make([]uint8, 320*200)}
	idx.Pix[0], idx.Pix[1] = 0xA, 0xB

	f, err := PackFrame(plan, idx)
	if err != nil {
		t.Fatal(err)
	}
	if f.Pixels[0] != 0xAB {
		t.Errorf("first pixel byte = $%02X, want $AB", f.Pixels[0])
	}
	if f.SCB[0] != 0x02 || f.SCB[199] != 0x02 {
		t.Errorf("SCBs = $%02X/$%02X, want $02 (320 mode, palette 2)", f.SCB[0], f.SCB[199])
	}
	if f.Palettes[2][5] != (shr.RGB12{R: 1, G: 2, B: 3}) {
		t.Error("palette not carried into frame")
	}
}

func TestPackFrame640(t *testing.T) {
	plan := Plan{Mode640: true}
	idx := Indexed{W: 640, H: 200, Pix: make([]uint8, 640*200)}
	idx.Pix[0], idx.Pix[1], idx.Pix[2], idx.Pix[3] = 3, 2, 1, 0
	f, err := PackFrame(plan, idx)
	if err != nil {
		t.Fatal(err)
	}
	if f.Pixels[0] != 0xE4 {
		t.Errorf("first pixel byte = $%02X, want $E4", f.Pixels[0])
	}
	if !shr.SCBIs640(f.SCB[0]) {
		t.Error("SCB should have the 640-mode bit set")
	}
}

func TestPackFrameDimensionMismatch(t *testing.T) {
	if _, err := PackFrame(Plan{}, Indexed{W: 640, H: 200, Pix: make([]uint8, 640*200)}); err == nil {
		t.Error("expected error: 640-wide pixels with a 320-mode plan")
	}
	if _, err := PackFrame(Plan{}, Indexed{W: 320, H: 100, Pix: make([]uint8, 320*100)}); err == nil {
		t.Error("expected error: wrong height")
	}
}

func TestRegistryLookups(t *testing.T) {
	if _, err := LookupTarget("no-such-target"); err == nil {
		t.Error("expected error for unknown target")
	}
	if _, err := LookupDitherer("no-such-dither"); err == nil {
		t.Error("expected error for unknown ditherer")
	}
	if _, err := LookupPlanner("no-such-planner"); err == nil {
		t.Error("expected error for unknown planner")
	}
}
