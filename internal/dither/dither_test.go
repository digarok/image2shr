package dither

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/internal/planner"
	"github.com/digarok/image2shr/shr"
)

func greyPlan() pipeline.Plan {
	var p pipeline.Plan
	p.Palettes[0] = planner.Grey16Palette()
	return p
}

func flat(v float32) *pix.LinearImage {
	img := pix.New(shr.Width320, shr.Height)
	for i := range img.Pix {
		img.Pix[i] = v
	}
	return img
}

func TestNoneOnExtremes(t *testing.T) {
	d, err := pipeline.LookupDitherer("none")
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		v    float32
		want uint8
	}{
		{0, 0},  // black → entry 0
		{1, 15}, // white → entry 15
	} {
		out, err := d.Apply(flat(tt.v), greyPlan(), pipeline.DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		for i, got := range out.Pix {
			if got != tt.want {
				t.Fatalf("value %v: pixel %d = %d, want %d", tt.v, i, got, tt.want)
			}
		}
	}
}

func TestNoneNearestGrey(t *testing.T) {
	// Linear value of grey level 11 ($0BBB → sRGB 11/15): every pixel at
	// exactly that value must map to index 11.
	v := pix.SRGBToLinear(11.0 / 15)
	d, _ := pipeline.LookupDitherer("none")
	out, err := d.Apply(flat(v), greyPlan(), pipeline.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if out.Pix[0] != 11 {
		t.Errorf("index = %d, want 11", out.Pix[0])
	}
}

// TestNearestIsPerceptual pins the matching metric. A mid-tone grey sits
// close to black in linear light (0.2 linear displays as ~sRGB 0.48), so
// linear-distance matching flips it to black — the black-speckle-on-white
// artifact. Perceptually it is near the middle and must go to white here.
func TestNearestIsPerceptual(t *testing.T) {
	var plan pipeline.Plan
	for i := range plan.Palettes[0] {
		plan.Palettes[0][i] = shr.RGB12{R: 15, G: 15, B: 15}
	}
	plan.Palettes[0][0] = shr.RGB12{} // black + white only
	d, _ := pipeline.LookupDitherer("none")
	out, err := d.Apply(flat(0.2), plan, pipeline.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if out.Pix[0] == 0 {
		t.Error("linear 0.2 (sRGB ~0.48) matched black, not white — matching is not perceptual")
	}
}

func TestDitherPreservesMeanLevel(t *testing.T) {
	// A flat image exactly between grey levels 7 and 8 (in linear light)
	// must dither to a mix of 7s and 8s whose mean linear value stays close
	// to the input — the whole point of dithering. Atkinson deliberately
	// drops a quarter of the error, so its tolerance is looser.
	l7 := pix.SRGBToLinear(7.0 / 15)
	l8 := pix.SRGBToLinear(8.0 / 15)
	target := (l7 + l8) / 2

	for _, tt := range []struct {
		name string
		tol  float64
	}{
		{"floyd-steinberg", 0.005},
		{"jarvis", 0.005},
		{"sierra", 0.005},
		{"atkinson", 0.02},
		{"bayer2", 0.02},
		{"bayer4", 0.02},
		{"bayer8", 0.02},
	} {
		d, err := pipeline.LookupDitherer(tt.name)
		if err != nil {
			t.Fatal(err)
		}
		out, err := d.Apply(flat(target), greyPlan(), pipeline.DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		var sum float64
		seen := map[uint8]int{}
		for _, idx := range out.Pix {
			if idx != 7 && idx != 8 {
				t.Fatalf("%s: unexpected index %d, want only 7 or 8", tt.name, idx)
			}
			seen[idx]++
			sum += float64(pix.SRGBToLinear(float32(idx) / 15))
		}
		if len(seen) != 2 {
			t.Errorf("%s: output uses only index %v — not dithering", tt.name, seen)
		}
		mean := sum / float64(len(out.Pix))
		if math.Abs(mean-float64(target)) > tt.tol {
			t.Errorf("%s: mean linear level = %v, want ~%v", tt.name, mean, target)
		}
	}
}

var allDithers = []string{
	"none", "floyd-steinberg", "atkinson", "jarvis", "sierra",
	"bayer2", "bayer4", "bayer8",
}

func TestDitherDeterministic(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	for y := 0; y < img.H; y++ {
		for x := 0; x < img.W; x++ {
			v := float32(x+y) / float32(img.W+img.H)
			img.Set(x, y, v, v, v)
		}
	}
	opt := pipeline.DefaultOptions()
	opt.Serpentine = true
	for _, name := range allDithers {
		d, err := pipeline.LookupDitherer(name)
		if err != nil {
			t.Fatal(err)
		}
		a, err := d.Apply(img, greyPlan(), opt)
		if err != nil {
			t.Fatal(err)
		}
		b, err := d.Apply(img, greyPlan(), opt)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(a.Pix, b.Pix) {
			t.Errorf("%s: same input produced different dither output", name)
		}
	}
}

func TestDitherStrengthZeroEqualsNone(t *testing.T) {
	img := pix.New(shr.Width320, shr.Height)
	for y := 0; y < img.H; y++ {
		for x := 0; x < img.W; x++ {
			v := float32(x) / float32(img.W)
			img.Set(x, y, v, v, v)
		}
	}
	opt := pipeline.DefaultOptions()
	opt.DitherStrength = 0
	nn, _ := pipeline.LookupDitherer("none")
	want, err := nn.Apply(img, greyPlan(), opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range allDithers[1:] {
		d, _ := pipeline.LookupDitherer(name)
		got, err := d.Apply(img, greyPlan(), opt)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.Pix, want.Pix) {
			t.Errorf("%s with strength 0 should equal nearest-color", name)
		}
	}
}

// TestBayerMatrix pins the recursive construction against the well-known
// 4×4 index matrix.
func TestBayerMatrix(t *testing.T) {
	want := [][]int{
		{0, 8, 2, 10},
		{12, 4, 14, 6},
		{3, 11, 1, 9},
		{15, 7, 13, 5},
	}
	if got := bayerMatrix(4); !reflect.DeepEqual(got, want) {
		t.Errorf("bayerMatrix(4) = %v, want %v", got, want)
	}
}

// TestBayerTiles: ordered dithering has no inter-pixel state, so a flat
// input must produce an exact n×n repeating tile.
func TestBayerTiles(t *testing.T) {
	l7 := pix.SRGBToLinear(7.0 / 15)
	l8 := pix.SRGBToLinear(8.0 / 15)
	for _, tt := range []struct {
		name string
		n    int
	}{{"bayer2", 2}, {"bayer4", 4}, {"bayer8", 8}} {
		d, _ := pipeline.LookupDitherer(tt.name)
		out, err := d.Apply(flat((l7+l8)/2), greyPlan(), pipeline.DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		for y := 0; y < out.H; y++ {
			for x := 0; x < out.W; x++ {
				if out.Pix[y*out.W+x] != out.Pix[(y%tt.n)*out.W+x%tt.n] {
					t.Fatalf("%s: pixel (%d,%d) breaks the %dx%d tile", tt.name, x, y, tt.n, tt.n)
				}
			}
		}
	}
}

func Test640Rejected(t *testing.T) {
	plan := greyPlan()
	plan.Mode640 = true
	d, _ := pipeline.LookupDitherer("floyd-steinberg")
	if _, err := d.Apply(flat(0), plan, pipeline.DefaultOptions()); !errors.Is(err, pipeline.ErrNotImplemented) {
		t.Errorf("640-mode dithering should be ErrNotImplemented, got %v", err)
	}
}
