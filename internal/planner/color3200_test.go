package planner

import (
	"testing"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

func plan3200(t *testing.T, img *pix.LinearImage) pipeline.Plan {
	t.Helper()
	plan, err := NewAdaptiveColor3200().Plan(img, pipeline.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// TestColor3200BeyondSixteenPalettes: 25 horizontal bands of 8 rows, each
// band holding 16 colors nobody else uses — 400 distinct colors, more than
// 16 shared palettes (256 entries) could ever hold. Lines whose ±windowRows
// training window stays inside their band must reproduce their 16 colors
// exactly; lines near a band edge legitimately compromise with the
// neighbors their window sees, so they are not asserted.
func TestColor3200BeyondSixteenPalettes(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	colorAt := func(x, y int) shr.RGB12 {
		b := y / 8
		return shr.RGB12{R: uint8(b % 16), G: uint8((b / 16) * 8), B: uint8(x / 20)}
	}
	for y := 0; y < shr.Height; y++ {
		for x := 0; x < shr.Width320; x++ {
			c := colorAt(x, y)
			r, g, b := nibColor(c.R, c.G, c.B)
			img.Set(x, y, r, g, b)
		}
	}

	plan := plan3200(t, img)
	if plan.LinePalettes == nil {
		t.Fatal("plan has no per-line palettes")
	}
	interior := 0
	for y := 0; y < shr.Height; y++ {
		if d := y % 8; d < windowRows || d >= 8-windowRows {
			continue // window crosses a band boundary
		}
		interior++
		for i := 0; i < 16; i++ {
			c := colorAt(i*20, y)
			if !paletteContains(plan.LinePalettes[y], c) {
				t.Fatalf("interior line %d color %v missing from its palette %v", y, c, plan.LinePalettes[y])
			}
		}
	}
	if interior == 0 {
		t.Fatal("test bug: no interior lines asserted")
	}

	unique := map[shr.RGB12]bool{}
	for y := range plan.LinePalettes {
		for _, c := range plan.LinePalettes[y] {
			unique[c] = true
		}
	}
	if len(unique) <= 256 {
		t.Errorf("only %d unique colors — no better than 16 shared palettes", len(unique))
	}
}

// TestColor3200PlanShape: 320 mode, no fill, SCB-side fields untouched.
func TestColor3200PlanShape(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	fillRect(img, 0, 0, shr.Width320, shr.Height, 0.4, 0.5, 0.6)
	plan := plan3200(t, img)
	if plan.Mode640 || plan.Fill {
		t.Error("plan must be 320 mode without fill")
	}
	for y, p := range plan.Line {
		if p != 0 {
			t.Fatalf("line %d SCB palette %d, want 0 (unused in Brooks)", y, p)
		}
	}
}

func TestColor3200Deterministic(t *testing.T) {
	run := func() [200][16]shr.RGB12 {
		img := pix.New(shr.Width320, shr.Height)
		for y := 0; y < shr.Height; y++ {
			for x := 0; x < shr.Width320; x++ {
				img.Set(x, y, float32(x)/319, float32(y)/199, float32(x+y)/518)
			}
		}
		return *plan3200(t, img).LinePalettes
	}
	if run() != run() {
		t.Error("same image produced different per-line palettes")
	}
}

// TestColor3200RejectsSCBMode: the format has no SCBs to plan.
func TestColor3200RejectsSCBMode(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	opt := pipeline.DefaultOptions()
	opt.SCBMode = "grouped"
	if _, err := NewAdaptiveColor3200().Plan(img, opt); err == nil {
		t.Error("planner accepted --scb-mode grouped")
	}
}
