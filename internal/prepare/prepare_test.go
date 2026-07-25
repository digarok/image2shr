package prepare

import (
	"math"
	"testing"

	"github.com/digarok/image2shr/internal/pix"
)

func almost(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-4 }

func TestCrop(t *testing.T) {
	src := pix.New(4, 4)
	src.Set(2, 1, 1, 0.5, 0.25)
	got, err := Crop(src, 2, 1, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.W != 2 || got.H != 2 {
		t.Fatalf("size = %dx%d, want 2x2", got.W, got.H)
	}
	if r, g, b := got.At(0, 0); !almost(r, 1) || !almost(g, 0.5) || !almost(b, 0.25) {
		t.Errorf("crop origin pixel = %v %v %v", r, g, b)
	}
}

func TestCropOutOfBounds(t *testing.T) {
	src := pix.New(4, 4)
	for _, c := range [][4]int{{-1, 0, 2, 2}, {0, 0, 5, 2}, {3, 3, 2, 2}, {0, 0, 0, 2}} {
		if _, err := Crop(src, c[0], c[1], c[2], c[3]); err == nil {
			t.Errorf("crop %v: expected error", c)
		}
	}
}

func TestResampleIdentity(t *testing.T) {
	// Same size, square pixels, stretch: values must survive exactly
	// (each dest pixel covers exactly one source pixel).
	src := pix.New(8, 8)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.Set(x, y, float32(x)/8, float32(y)/8, 0.5)
		}
	}
	got, err := Resample(src, 8, 8, 1.0, FitStretch)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			gr, gg, gb := got.At(x, y)
			sr, sg, sb := src.At(x, y)
			if !almost(gr, sr) || !almost(gg, sg) || !almost(gb, sb) {
				t.Fatalf("pixel (%d,%d) changed: got %v %v %v want %v %v %v", x, y, gr, gg, gb, sr, sg, sb)
			}
		}
	}
}

func TestResampleDownscaleAverages(t *testing.T) {
	// 2×1 black+white downscaled to 1×1 must average to 0.5 in linear light.
	src := pix.New(2, 1)
	src.Set(0, 0, 1, 1, 1)
	got, err := Resample(src, 1, 1, 1.0, FitStretch)
	if err != nil {
		t.Fatal(err)
	}
	if r, _, _ := got.At(0, 0); !almost(r, 0.5) {
		t.Errorf("average = %v, want 0.5", r)
	}
}

func TestResampleContainLetterbox(t *testing.T) {
	// A square white source into a 320x200 frame with par=1.2 (display
	// 320x240): contain scales it to 240x240 display units => 240 px wide,
	// centered with 40 px black bars left and right, full height.
	src := pix.New(10, 10)
	for i := range src.Pix {
		src.Pix[i] = 1
	}
	got, err := Resample(src, 320, 200, 1.2, FitContain)
	if err != nil {
		t.Fatal(err)
	}
	if r, _, _ := got.At(10, 100); r != 0 {
		t.Errorf("left letterbox pixel = %v, want black", r)
	}
	if r, _, _ := got.At(160, 100); !almost(r, 1) {
		t.Errorf("center pixel = %v, want white", r)
	}
	if r, _, _ := got.At(160, 0); !almost(r, 1) {
		t.Errorf("top center pixel = %v, want white (full height)", r)
	}
	if r, _, _ := got.At(310, 100); r != 0 {
		t.Errorf("right letterbox pixel = %v, want black", r)
	}
}

func TestResampleCoverFills(t *testing.T) {
	// Cover mode must leave no black anywhere for a white source.
	src := pix.New(10, 10)
	for i := range src.Pix {
		src.Pix[i] = 1
	}
	got, err := Resample(src, 320, 200, 1.2, FitCover)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range [][2]int{{0, 0}, {319, 0}, {0, 199}, {319, 199}, {160, 100}} {
		if r, _, _ := got.At(p[0], p[1]); !almost(r, 1) {
			t.Errorf("cover pixel %v = %v, want white", p, r)
		}
	}
}

func TestResampleNone(t *testing.T) {
	// fit=none: 1:1 pixels, centered. 4x2 source in 8x4 dest => offset (2,1).
	src := pix.New(4, 2)
	src.Set(0, 0, 1, 0, 0)
	got, err := Resample(src, 8, 4, 1.2, FitNone)
	if err != nil {
		t.Fatal(err)
	}
	if r, _, _ := got.At(2, 1); !almost(r, 1) {
		t.Errorf("source origin should land at (2,1), got %v", r)
	}
	if r, _, _ := got.At(0, 0); r != 0 {
		t.Errorf("border should be black, got %v", r)
	}
}

func TestResampleUnknownFit(t *testing.T) {
	if _, err := Resample(pix.New(1, 1), 2, 2, 1, "bogus"); err == nil {
		t.Error("expected error for unknown fit mode")
	}
}

func TestPixelAspect(t *testing.T) {
	if got := PixelAspect(320, 200, "correct"); !almost(float32(got), 1.2) {
		t.Errorf("320x200 par = %v, want 1.2", got)
	}
	if got := PixelAspect(640, 200, "correct"); !almost(float32(got), 2.4) {
		t.Errorf("640x200 par = %v, want 2.4", got)
	}
	if got := PixelAspect(320, 200, "ignore"); got != 1.0 {
		t.Errorf("ignore par = %v, want 1.0", got)
	}
}

func TestAdjustmentsIdentity(t *testing.T) {
	img := pix.New(1, 1)
	img.Set(0, 0, 0.3, 0.6, 0.9)
	Adjustments{Gamma: 1, Brightness: 0, Contrast: 1, Saturation: 1}.Apply(img)
	if r, g, b := img.At(0, 0); !almost(r, 0.3) || !almost(g, 0.6) || !almost(b, 0.9) {
		t.Errorf("identity adjustments changed pixel: %v %v %v", r, g, b)
	}
}

func TestAdjustments(t *testing.T) {
	img := pix.New(1, 1)
	img.Set(0, 0, 0.25, 0.25, 0.25)
	Adjustments{Gamma: 2, Brightness: 0, Contrast: 1, Saturation: 1}.Apply(img)
	if r, _, _ := img.At(0, 0); !almost(r, 0.5) {
		t.Errorf("gamma 2 on 0.25 = %v, want 0.5 (0.25^(1/2))", r)
	}

	img.Set(0, 0, 0.5, 0.5, 0.5)
	Adjustments{Gamma: 1, Brightness: 0.2, Contrast: 1, Saturation: 1}.Apply(img)
	if r, _, _ := img.At(0, 0); !almost(r, 0.7) {
		t.Errorf("brightness +0.2 on 0.5 = %v, want 0.7", r)
	}

	img.Set(0, 0, 0.75, 0.75, 0.75)
	Adjustments{Gamma: 1, Brightness: 0, Contrast: 2, Saturation: 1}.Apply(img)
	if r, _, _ := img.At(0, 0); !almost(r, 1.0) {
		t.Errorf("contrast 2 on 0.75 = %v, want 1.0 (clamped)", r)
	}

	// Saturation 0 turns pure red into its rec709 luma grey.
	img.Set(0, 0, 1, 0, 0)
	Adjustments{Gamma: 1, Brightness: 0, Contrast: 1, Saturation: 0}.Apply(img)
	r, g, b := img.At(0, 0)
	if !almost(r, 0.2126) || !almost(g, 0.2126) || !almost(b, 0.2126) {
		t.Errorf("desaturated red = %v %v %v, want 0.2126 grey", r, g, b)
	}
}
