package planner

import (
	"testing"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

// nibColor returns the linear-light value of an RGB444 color, i.e. exactly
// what a prepared image holds when the source already sits on the 4-bit grid.
func nibColor(r, g, b uint8) (fr, fg, fb float32) {
	return pix.SRGBToLinear(float32(r) / 15),
		pix.SRGBToLinear(float32(g) / 15),
		pix.SRGBToLinear(float32(b) / 15)
}

func fillRect(img *pix.LinearImage, x0, y0, w, h int, r, g, b float32) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			img.Set(x, y, r, g, b)
		}
	}
}

func paletteContains(pal [16]shr.RGB12, c shr.RGB12) bool {
	for _, p := range pal {
		if p == c {
			return true
		}
	}
	return false
}

// TestColor16ExactWhenFewColors: an image with ≤16 distinct RGB444 colors
// must get every one of them back exactly — zero error is achievable.
func TestColor16ExactWhenFewColors(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	colors := []shr.RGB12{
		{R: 0, G: 0, B: 0}, {R: 15, G: 0, B: 0}, {R: 0, G: 15, B: 0},
		{R: 0, G: 0, B: 15}, {R: 15, G: 15, B: 0}, {R: 8, G: 4, B: 12},
		{R: 15, G: 15, B: 15}, {R: 5, G: 5, B: 5},
	}
	for i, c := range colors {
		r, g, b := nibColor(c.R, c.G, c.B)
		fillRect(img, i*40, 0, 40, shr.Height, r, g, b)
	}

	pal := bestPalette16(img)
	for _, c := range colors {
		if !paletteContains(pal, c) {
			t.Errorf("palette %v missing exact source color %v", pal, c)
		}
	}
}

// TestColor16SmallRegionSurvives is the psychovisual point of the planner:
// 400 saturated red pixels among 64,000 greys must keep a palette entry.
// Every grey stripe has ten times as many pixels as the red patch, so a
// popularity histogram would drop the red; perceptual clustering must not,
// because the red sits far from every grey in Oklab.
func TestColor16SmallRegionSurvives(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	for n := 0; n < 16; n++ { // 16 grey stripes + red = 17 candidates for 16 slots
		r, g, b := nibColor(uint8(n), uint8(n), uint8(n))
		fillRect(img, n*20, 0, 20, shr.Height, r, g, b)
	}
	red := shr.RGB12{R: 15, G: 0, B: 0}
	rr, rg, rb := nibColor(red.R, red.G, red.B)
	fillRect(img, 100, 100, 20, 20, rr, rg, rb)

	pal := bestPalette16(img)
	if !paletteContains(pal, red) {
		t.Fatalf("palette %v lost the small red region — reduction is behaving like a histogram", pal)
	}
}

func TestColor16Deterministic(t *testing.T) {
	make16 := func() [16]shr.RGB12 {
		img := pix.New(shr.Width320, shr.Height)
		for y := 0; y < shr.Height; y++ {
			for x := 0; x < shr.Width320; x++ {
				img.Set(x, y, float32(x)/319, float32(y)/199, float32(x+y)/518)
			}
		}
		return bestPalette16(img)
	}
	a, b := make16(), make16()
	if a != b {
		t.Errorf("same image produced different palettes:\n%v\n%v", a, b)
	}
}

// TestColor16PlanShape: single palette in slot 0, all lines on it, 320 mode.
func TestColor16PlanShape(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	fillRect(img, 0, 0, shr.Width320, shr.Height, 0.5, 0.2, 0.1)

	plan, err := NewAdaptiveColor16().Plan(img, pipeline.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Mode640 || plan.Fill {
		t.Error("plan must be 320 mode without fill")
	}
	for y, p := range plan.Line {
		if p != 0 {
			t.Fatalf("line %d assigned palette %d, want 0", y, p)
		}
	}
	// A single-color image: entry padding must duplicate a real color, not
	// introduce black.
	for i, c := range plan.Palettes[0] {
		if c != plan.Palettes[0][0] {
			t.Errorf("entry %d = %v, want duplicate of %v (single-color image)", i, c, plan.Palettes[0][0])
		}
	}
}
