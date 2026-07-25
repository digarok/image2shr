package cli

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Golden round-trip: fixture.png → convert → .shr (byte-exact) → preview →
// PNG (pixel-exact). Regenerate after an INTENTIONAL output change with:
//
//	go test ./internal/cli -run Golden -update
//
// The fixture itself comes from tools/genfixture and is committed.
var update = flag.Bool("update", false, "rewrite golden files")

const (
	fixturePNG = "testdata/fixture.png"
	goldenSHR  = "testdata/golden.shr"
	goldenPNG  = "testdata/golden.png"
)

func TestGoldenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	outSHR := filepath.Join(dir, "out.shr")
	outPNG := filepath.Join(dir, "out.png")

	// Default flags: shr320-grey16, floyd-steinberg, contain, aspect correct.
	code, stdout, stderr := run(t, nil, "convert", "-o", outSHR, fixturePNG)
	if code != 0 {
		t.Fatalf("convert exit %d, stderr: %s", code, stderr)
	}
	if len(stdout) != 0 {
		t.Fatalf("convert stdout not empty: %d bytes", len(stdout))
	}
	code, _, stderr = run(t, nil, "preview", "-o", outPNG, outSHR)
	if code != 0 {
		t.Fatalf("preview exit %d, stderr: %s", code, stderr)
	}

	gotSHR, err := os.ReadFile(outSHR)
	if err != nil {
		t.Fatal(err)
	}
	gotPNG, err := os.ReadFile(outPNG)
	if err != nil {
		t.Fatal(err)
	}

	if *update {
		if err := os.WriteFile(goldenSHR, gotSHR, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPNG, gotPNG, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Log("golden files updated")
		return
	}

	wantSHR, err := os.ReadFile(goldenSHR)
	if err != nil {
		t.Fatalf("%v (run with -update to create golden files)", err)
	}
	if !bytes.Equal(gotSHR, wantSHR) {
		t.Errorf(".shr output differs from %s (byte-exact comparison); "+
			"if the change is intentional, rerun with -update", goldenSHR)
	}

	// Compare the preview by decoded pixels, not PNG bytes, so a Go stdlib
	// encoder change can't break the test spuriously.
	want := decodePNG(t, goldenPNG)
	got := decodePNGBytes(t, gotPNG)
	if !want.Bounds().Eq(got.Bounds()) {
		t.Fatalf("preview bounds %v != golden %v", got.Bounds(), want.Bounds())
	}
	b := want.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if want.At(x, y) != got.At(x, y) {
				t.Fatalf("preview pixel (%d,%d) differs from golden; "+
					"if intentional, rerun with -update", x, y)
			}
		}
	}
}

func decodePNG(t *testing.T, path string) image.Image {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to create golden files)", err)
	}
	return decodePNGBytes(t, data)
}

func decodePNGBytes(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return img
}
