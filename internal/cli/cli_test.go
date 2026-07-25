package cli

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/digarok/image2shr/shr"
)

// run invokes the CLI as a process would, returning exit code and streams.
func run(t *testing.T, stdin []byte, args ...string) (code int, stdout, stderr []byte) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code = Main(args, &out, &errBuf, bytes.NewReader(stdin))
	return code, out.Bytes(), errBuf.Bytes()
}

// testPNG writes a small greyscale gradient PNG and returns its path.
func testPNG(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 64, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 64; x++ {
			v := uint8(x * 255 / 63)
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	path := filepath.Join(t.TempDir(), "in.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVersion(t *testing.T) {
	code, out, _ := run(t, nil, "version")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.HasPrefix(string(out), "image2shr ") {
		t.Errorf("output %q", out)
	}
}

func TestTargetsListsGrey16(t *testing.T) {
	code, out, _ := run(t, nil, "targets")
	if code != 0 {
		t.Fatalf("exit %d, want 0", code)
	}
	if !strings.Contains(string(out), "shr320-grey16") {
		t.Errorf("targets output missing shr320-grey16: %q", out)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, _ := run(t, nil, "frobnicate")
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestNoCommand(t *testing.T) {
	code, _, _ := run(t, nil)
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestConvertUsageErrors(t *testing.T) {
	in := testPNG(t)
	cases := [][]string{
		{"convert"},                               // no input
		{"convert", "--dither", "wavelet", in},    // unknown dither
		{"convert", "--fit", "zoom", in},          // bad fit
		{"convert", "--aspect", "maybe", in},      // bad aspect
		{"convert", "--crop", "1,2,3", in},        // malformed crop
		{"convert", "--dither-strength", "2", in}, // out of range
		{"convert", "--gamma", "0", in},           // gamma must be > 0
		{"convert", "--luma", "hsl", in},          // bad luma
		{"convert", "--scb-mode", "chaotic", in},  // bad scb-mode
		{"convert", "-t", "shr640-nope", in},      // unknown target
		{"convert", "--format", "gif", in},        // unknown format
		{"convert", "--json", "-o", "-", in},      // stdout conflict
		{"convert", "--sidecar", "-o", "-", in},   // sidecar needs a file
		{"convert", "-"},                          // stdin requires -o
	}
	for _, args := range cases {
		code, _, stderr := run(t, nil, args...)
		if code != 2 {
			t.Errorf("%v: exit %d, want 2 (stderr: %s)", args, code, stderr)
		}
	}
}

func TestConvertStubDitherIsRuntimeError(t *testing.T) {
	in := testPNG(t)
	out := filepath.Join(t.TempDir(), "out.shr")
	code, _, stderr := run(t, nil, "convert", "--dither", "atkinson", "-o", out, in)
	if code != 1 {
		t.Errorf("exit %d, want 1 (stub algorithm is a runtime failure)", code)
	}
	if !strings.Contains(string(stderr), "not yet implemented") {
		t.Errorf("stderr %q should mention not yet implemented", stderr)
	}
}

func TestConvertEndToEnd(t *testing.T) {
	in := testPNG(t)
	out := filepath.Join(t.TempDir(), "out.shr")
	code, stdout, stderr := run(t, nil, "convert", "-o", out, in)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if len(stdout) != 0 {
		t.Errorf("stdout must be empty without --json, got %d bytes", len(stdout))
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != shr.FrameSize {
		t.Fatalf("output = %d bytes, want %d", len(data), shr.FrameSize)
	}
	frame, err := shr.DecodeRaw(data)
	if err != nil {
		t.Fatal(err)
	}
	for y, scb := range frame.SCB {
		if scb != 0 {
			t.Fatalf("SCB[%d] = $%02X, want $00", y, scb)
		}
	}
	if frame.Palettes[0][15] != (shr.RGB12{R: 15, G: 15, B: 15}) {
		t.Error("palette 0 entry 15 should be $0FFF")
	}
}

func TestConvertDeterministic(t *testing.T) {
	in := testPNG(t)
	dir := t.TempDir()
	a := filepath.Join(dir, "a.shr")
	b := filepath.Join(dir, "b.shr")
	for _, out := range []string{a, b} {
		if code, _, stderr := run(t, nil, "convert", "--serpentine", "-o", out, in); code != 0 {
			t.Fatalf("exit %d, stderr: %s", code, stderr)
		}
	}
	da, _ := os.ReadFile(a)
	db, _ := os.ReadFile(b)
	if !bytes.Equal(da, db) {
		t.Error("same input + flags produced different bytes")
	}
}

func TestConvertStdinStdout(t *testing.T) {
	in := testPNG(t)
	pngBytes, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := run(t, pngBytes, "convert", "-o", "-", "-")
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	if len(stdout) != shr.FrameSize {
		t.Fatalf("stdout = %d bytes, want %d (binary payload)", len(stdout), shr.FrameSize)
	}
}

func TestConvertJSON(t *testing.T) {
	in := testPNG(t)
	out := filepath.Join(t.TempDir(), "out.shr")
	code, stdout, stderr := run(t, nil, "convert", "--json", "--seed", "7", "-o", out, in)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	var rep convertReport
	if err := json.Unmarshal(stdout, &rep); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, stdout)
	}
	if rep.InputWidth != 64 || rep.InputHeight != 40 {
		t.Errorf("input dims %dx%d, want 64x40", rep.InputWidth, rep.InputHeight)
	}
	if rep.Target != "shr320-grey16" || rep.OutputSize != shr.FrameSize {
		t.Errorf("target/size = %s/%d", rep.Target, rep.OutputSize)
	}
	if rep.ProDOS.FileTypeHex != "$C1" || rep.ProDOS.AuxTypeHex != "$0000" {
		t.Errorf("prodos = %+v", rep.ProDOS)
	}
	if len(rep.LinePalettes) != 200 {
		t.Errorf("line_palettes has %d entries, want 200", len(rep.LinePalettes))
	}
	if len(rep.PalettesUsed) != 1 || rep.PalettesUsed[0].Index != 0 {
		t.Errorf("palettes_used = %+v", rep.PalettesUsed)
	}
	if len(rep.Warnings) != 1 || !strings.Contains(rep.Warnings[0], "--seed") {
		t.Errorf("warnings = %v, want the --seed no-effect warning", rep.Warnings)
	}
}

func TestConvertSidecarAndPreviewPNG(t *testing.T) {
	in := testPNG(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "out.shr")
	prev := filepath.Join(dir, "prev.png")
	code, _, stderr := run(t, nil, "convert", "--sidecar", "--preview-png", prev, "-o", out, in)
	if code != 0 {
		t.Fatalf("exit %d, stderr: %s", code, stderr)
	}
	side, err := os.ReadFile(out + ".meta.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(side), `"$C1"`) {
		t.Errorf("sidecar missing file type: %s", side)
	}
	pf, err := os.Open(prev)
	if err != nil {
		t.Fatal(err)
	}
	defer pf.Close()
	img, err := png.Decode(pf)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != 640 || img.Bounds().Dy() != 200 {
		t.Errorf("preview PNG = %v, want 640x200", img.Bounds())
	}
}

func TestPreviewAndInspect(t *testing.T) {
	in := testPNG(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "out.shr")
	if code, _, stderr := run(t, nil, "convert", "-o", out, in); code != 0 {
		t.Fatalf("convert failed: %s", stderr)
	}

	// preview
	pngOut := filepath.Join(dir, "out.png")
	code, stdout, stderr := run(t, nil, "preview", "-o", pngOut, out)
	if code != 0 {
		t.Fatalf("preview exit %d, stderr: %s", code, stderr)
	}
	if len(stdout) != 0 {
		t.Error("preview to a file must leave stdout empty")
	}

	// preview --scale
	code, _, _ = run(t, nil, "preview", "--scale", "2", "-o", "-", out)
	if code != 0 {
		t.Fatalf("preview --scale exit %d", code)
	}

	// inspect (human)
	code, stdout, stderr = run(t, nil, "inspect", out)
	if code != 0 {
		t.Fatalf("inspect exit %d, stderr: %s", code, stderr)
	}
	if !strings.Contains(string(stdout), "200 in 320 mode") {
		t.Errorf("inspect output: %s", stdout)
	}

	// inspect --json
	code, stdout, _ = run(t, nil, "inspect", "--json", out)
	if code != 0 {
		t.Fatalf("inspect --json exit %d", code)
	}
	var rep inspectReport
	if err := json.Unmarshal(stdout, &rep); err != nil {
		t.Fatalf("inspect --json invalid: %v", err)
	}
	if rep.Lines320 != 200 || rep.Lines640 != 0 {
		t.Errorf("lines = %d/%d, want 200/0", rep.Lines320, rep.Lines640)
	}
	if rep.UniqueColors < 2 || rep.UniqueColors > 16 {
		t.Errorf("unique colors = %d, want within 2..16", rep.UniqueColors)
	}
}

func TestPreviewRejectsBadFile(t *testing.T) {
	bad := filepath.Join(t.TempDir(), "bad.shr")
	if err := os.WriteFile(bad, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := run(t, nil, "preview", "-o", "-", bad)
	if code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(string(stderr), "32768") {
		t.Errorf("stderr should explain the size requirement: %s", stderr)
	}
}
