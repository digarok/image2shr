package planner

import (
	"testing"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/shr"
)

func planWith(t *testing.T, img *pix.LinearImage, mode string) pipeline.Plan {
	t.Helper()
	opt := pipeline.DefaultOptions()
	opt.SCBMode = mode
	plan, err := NewAdaptiveColor256().Plan(img, opt)
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

// gradientImage sweeps hue and lightness down the screen so every band has
// distinct content.
func gradientImage() *pix.LinearImage {
	img := pix.New(shr.Width320, shr.Height)
	for y := 0; y < shr.Height; y++ {
		for x := 0; x < shr.Width320; x++ {
			img.Set(x, y, float32(y)/199, float32(x)/319, 1-float32(y)/199)
		}
	}
	return img
}

// TestColor256BandedShape: the contiguous strategies must produce exactly 16
// bands in top-to-bottom palette order, 320 mode, no fill.
func TestColor256BandedShape(t *testing.T) {
	img := gradientImage()
	for _, mode := range []string{"banded", "grouped"} {
		plan := planWith(t, img, mode)
		if plan.Mode640 || plan.Fill {
			t.Errorf("%s: plan must be 320 mode without fill", mode)
		}
		var seen [16]bool
		prev := uint8(0)
		for y, p := range plan.Line {
			if p < prev {
				t.Fatalf("%s: line %d palette %d after palette %d — bands must be contiguous top to bottom", mode, y, p, prev)
			}
			prev = p
			seen[p] = true
		}
		for p, ok := range seen {
			if !ok {
				t.Errorf("%s: palette %d unused — want all 16 bands non-empty", mode, p)
			}
		}
	}
}

// TestColor256StripesExactPerBand: 32 distinct colors in horizontal stripes
// exceed one palette but not sixteen; every line must find its exact stripe
// color in its assigned palette.
func TestColor256StripesExactPerBand(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	stripe := func(i int) shr.RGB12 {
		return shr.RGB12{R: uint8(i % 16), G: uint8((i / 2) % 16), B: uint8(15 - i%16)}
	}
	for y := 0; y < shr.Height; y++ {
		c := stripe(y * 32 / shr.Height)
		r, g, b := nibColor(c.R, c.G, c.B)
		fillRect(img, 0, y, shr.Width320, 1, r, g, b)
	}
	for _, mode := range []string{"banded", "grouped"} {
		plan := planWith(t, img, mode)
		for y := 0; y < shr.Height; y++ {
			want := stripe(y * 32 / shr.Height)
			if !paletteContains(plan.Palettes[plan.Line[y]], want) {
				t.Fatalf("%s: line %d color %v missing from its palette %d: %v",
					mode, y, want, plan.Line[y], plan.Palettes[plan.Line[y]])
			}
		}
	}
}

// TestColor256PerLineSplitsInterleaved is the point of the fully dynamic
// mode: even lines hold 16 warm colors, odd lines 16 cool ones. Contiguous
// bands see 32 colors everywhere and must compromise; per-line clustering
// can give each line family its own exact palette.
func TestColor256PerLineSplitsInterleaved(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	warm := func(i int) shr.RGB12 { return shr.RGB12{R: 15, G: uint8(i), B: 0} }
	cool := func(i int) shr.RGB12 { return shr.RGB12{R: 0, G: uint8(i), B: 15} }
	colorAt := func(x, y int) shr.RGB12 {
		if y%2 == 0 {
			return warm(x / 20)
		}
		return cool(x / 20)
	}
	for y := 0; y < shr.Height; y++ {
		for x := 0; x < shr.Width320; x++ {
			c := colorAt(x, y)
			r, g, b := nibColor(c.R, c.G, c.B)
			img.Set(x, y, r, g, b)
		}
	}

	plan := planWith(t, img, "per-line")
	for y := 0; y < shr.Height; y++ {
		for i := 0; i < 16; i++ {
			c := colorAt(i*20, y)
			if !paletteContains(plan.Palettes[plan.Line[y]], c) {
				t.Fatalf("line %d color %v missing from its palette %d: %v — per-line mode failed to separate the interleaved sets",
					y, c, plan.Line[y], plan.Palettes[plan.Line[y]])
			}
		}
	}
}

// TestColor256SingleMode: --scb-mode single degenerates to the color16 plan.
func TestColor256SingleMode(t *testing.T) {
	img := gradientImage()
	plan := planWith(t, img, "single")
	for y, p := range plan.Line {
		if p != 0 {
			t.Fatalf("line %d assigned palette %d, want 0", y, p)
		}
	}
	if plan.Palettes[0] != bestPalette16(img) {
		t.Error("single mode palette differs from bestPalette16")
	}
}

func TestColor256Deterministic(t *testing.T) {
	for _, mode := range []string{"banded", "grouped", "per-line"} {
		a := planWith(t, gradientImage(), mode)
		b := planWith(t, gradientImage(), mode)
		if a != b {
			t.Errorf("%s: same image produced different plans", mode)
		}
	}
}

// TestSinglePaletteGuard: the single-palette planners must reject the
// multi-palette modes instead of silently ignoring them.
func TestSinglePaletteGuard(t *testing.T) {
	img := gradientImage()
	opt := pipeline.DefaultOptions()
	opt.SCBMode = "grouped"
	if _, err := NewFixed("fixed-test", Grey16Palette()).Plan(img, opt); err == nil {
		t.Error("Fixed accepted --scb-mode grouped")
	}
	if _, err := NewAdaptiveColor16().Plan(img, opt); err == nil {
		t.Error("AdaptiveColor16 accepted --scb-mode grouped")
	}
	opt.SCBMode = "bogus"
	if _, err := NewAdaptiveColor256().Plan(img, opt); err == nil {
		t.Error("AdaptiveColor256 accepted --scb-mode bogus")
	}
}
