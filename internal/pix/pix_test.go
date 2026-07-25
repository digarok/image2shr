package pix

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestSRGBLinearRoundTrip(t *testing.T) {
	for _, v := range []float32{0, 0.001, 0.04, 0.2, 0.5, 0.9, 1} {
		back := LinearToSRGB(SRGBToLinear(v))
		if math.Abs(float64(back-v)) > 1e-5 {
			t.Errorf("round trip %v -> %v", v, back)
		}
	}
	// sRGB mid-grey 0.5 is ~0.2140 in linear light.
	if l := SRGBToLinear(0.5); math.Abs(float64(l)-0.2140) > 0.001 {
		t.Errorf("SRGBToLinear(0.5) = %v, want ~0.2140", l)
	}
}

func TestFromImage(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{255, 255, 255, 255})
	src.Set(1, 0, color.RGBA{0, 0, 0, 255})
	li := FromImage(src)
	if li.W != 2 || li.H != 1 {
		t.Fatalf("size = %dx%d, want 2x1", li.W, li.H)
	}
	if r, g, b := li.At(0, 0); r != 1 || g != 1 || b != 1 {
		t.Errorf("white pixel = %v %v %v, want 1 1 1", r, g, b)
	}
	if r, _, _ := li.At(1, 0); r != 0 {
		t.Errorf("black pixel r = %v, want 0", r)
	}
}

func TestLuma(t *testing.T) {
	w, err := LumaOf("rec709")
	if err != nil {
		t.Fatal(err)
	}
	if y := w.Apply(1, 1, 1); math.Abs(float64(y)-1) > 1e-4 {
		t.Errorf("luma of white = %v, want 1 (weights must sum to 1)", y)
	}
	if _, err := LumaOf("bogus"); err == nil {
		t.Error("expected error for unknown luma name")
	}
}

func TestToGrey(t *testing.T) {
	li := New(1, 1)
	li.Set(0, 0, 1, 0, 0) // pure red
	w, _ := LumaOf("rec709")
	g := ToGrey(li, w)
	r, gr, b := g.At(0, 0)
	if r != gr || gr != b {
		t.Errorf("grey pixel channels differ: %v %v %v", r, gr, b)
	}
	if math.Abs(float64(r)-0.2126) > 1e-4 {
		t.Errorf("grey of pure red = %v, want 0.2126", r)
	}
}
