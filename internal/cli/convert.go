package cli

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
	"github.com/digarok/image2shr/internal/prepare"
	"github.com/digarok/image2shr/internal/source"
	"github.com/digarok/image2shr/internal/writer"
	"github.com/digarok/image2shr/shr"

	// Register targets and ditherers.
	_ "github.com/digarok/image2shr/internal/dither"
	_ "github.com/digarok/image2shr/internal/target"
)

const convertUsage = `Convert an image (png, jpeg, gif, bmp) to an Apple IIgs Super Hi-Res file.

Usage:
  image2shr convert [flags] <input>

The input "-" reads from stdin. Without -o, the output lands next to the
input with the extension replaced by .shr (stdin input requires -o).

Examples:
  image2shr convert photo.png
  image2shr convert -o title.shr -t shr320-grey16 --dither floyd-steinberg title.jpg
  image2shr convert -o title.shr -t shr320-color256 --scb-mode per-line title.jpg
  image2shr convert --dither none --fit cover --aspect ignore -o out.shr in.bmp
  image2shr convert --preview-png check.png --sidecar -o out.shr in.png
  image2shr convert --json -o out.shr in.png       # machine-readable report on stdout
  cat in.png | image2shr convert -o - - > out.shr  # fully piped
`

// convertConfig is everything cmdConvert parses; runConvert executes it.
// Split so tests (and the golden harness) can run conversions directly.
type convertConfig struct {
	Input      string
	Output     string
	PreviewPNG string
	Sidecar    bool
	JSON       bool
	Verbose    bool
	Quiet      bool
	Opt        pipeline.Options
}

func cmdConvert(e *env, args []string) error {
	fs := newFlagSet(e, "convert", convertUsage)
	def := pipeline.DefaultOptions()

	cfg := convertConfig{Opt: def}
	var cropSpec string
	fs.StringVar(&cfg.Output, "output", "", "output file, \"-\" for stdout")
	fs.StringVar(&cfg.Output, "o", "", "alias for --output")
	fs.StringVar(&cfg.Opt.Target, "target", def.Target, "conversion target (see \"image2shr targets\")")
	fs.StringVar(&cfg.Opt.Target, "t", def.Target, "alias for --target")
	fs.StringVar(&cfg.Opt.Format, "format", def.Format, "output container: raw, packed, apf, brooks")
	fs.StringVar(&cfg.Opt.Dither, "dither", def.Dither, "none, floyd-steinberg, atkinson, jarvis, sierra, bayer2/4/8")
	fs.Float64Var(&cfg.Opt.DitherStrength, "dither-strength", def.DitherStrength, "error diffusion strength, 0.0-1.0")
	fs.BoolVar(&cfg.Opt.Serpentine, "serpentine", false, "serpentine scan for error diffusion")
	fs.StringVar(&cfg.Opt.SCBMode, "scb-mode", def.SCBMode, "auto, single, banded, grouped, per-line")
	fs.StringVar(&cfg.Opt.Palette, "palette", "", "named builtin or a palette file")
	fs.StringVar(&cfg.Opt.Fit, "fit", def.Fit, "contain, cover, stretch, none")
	fs.StringVar(&cfg.Opt.Aspect, "aspect", def.Aspect, "correct, ignore")
	fs.StringVar(&cropSpec, "crop", "", "crop rectangle X,Y,W,H in source pixels")
	fs.Float64Var(&cfg.Opt.Gamma, "gamma", def.Gamma, "gamma adjustment (1.0 = none)")
	fs.Float64Var(&cfg.Opt.Brightness, "brightness", def.Brightness, "brightness offset (0.0 = none)")
	fs.Float64Var(&cfg.Opt.Contrast, "contrast", def.Contrast, "contrast factor (1.0 = none)")
	fs.Float64Var(&cfg.Opt.Saturation, "saturation", def.Saturation, "saturation factor (1.0 = none)")
	fs.StringVar(&cfg.Opt.Luma, "luma", def.Luma, "rec601, rec709, average — greyscale weights")
	fs.StringVar(&cfg.PreviewPNG, "preview-png", "", "also write a round-tripped PNG of the result")
	fs.BoolVar(&cfg.Sidecar, "sidecar", false, "write ProDOS type/auxtype metadata to <output>.meta.json")
	fs.Int64Var(&cfg.Opt.Seed, "seed", 0, "seed for any randomized step (none yet)")
	fs.BoolVar(&cfg.JSON, "json", false, "machine-readable result on stdout")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "log pipeline stages to stderr")
	fs.BoolVar(&cfg.Verbose, "v", false, "alias for --verbose")
	fs.BoolVar(&cfg.Quiet, "quiet", false, "suppress warnings on stderr")
	fs.BoolVar(&cfg.Quiet, "q", false, "alias for --quiet")

	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usagef("convert needs exactly one input file (got %d); see --help", len(pos))
	}
	cfg.Input = pos[0]

	if cropSpec != "" {
		var c pipeline.CropRect
		if n, err := fmt.Sscanf(cropSpec, "%d,%d,%d,%d", &c.X, &c.Y, &c.W, &c.H); n != 4 || err != nil {
			return usagef("--crop must be X,Y,W,H (got %q)", cropSpec)
		}
		cfg.Opt.Crop = &c
	}
	if err := validateConvert(&cfg); err != nil {
		return err
	}
	return runConvert(e, cfg)
}

// validateConvert checks every flag value that can be judged without doing
// work, so bad invocations exit 2 before any file is touched.
func validateConvert(cfg *convertConfig) error {
	o := &cfg.Opt
	if _, err := pipeline.LookupTarget(o.Target); err != nil {
		return usageError{err}
	}
	if _, err := writer.Lookup(o.Format); err != nil {
		return usageError{err}
	}
	if _, err := pipeline.LookupDitherer(o.Dither); err != nil {
		return usageError{err}
	}
	if o.DitherStrength < 0 || o.DitherStrength > 1 {
		return usagef("--dither-strength %v out of range 0.0-1.0", o.DitherStrength)
	}
	switch o.SCBMode {
	case "auto", "single", "banded", "grouped", "per-line":
	default:
		return usagef("--scb-mode %q invalid (valid: auto, single, banded, grouped, per-line)", o.SCBMode)
	}
	switch o.Fit {
	case prepare.FitContain, prepare.FitCover, prepare.FitStretch, prepare.FitNone:
	default:
		return usagef("--fit %q invalid (valid: contain, cover, stretch, none)", o.Fit)
	}
	switch o.Aspect {
	case "correct", "ignore":
	default:
		return usagef("--aspect %q invalid (valid: correct, ignore)", o.Aspect)
	}
	if _, err := pix.LumaOf(o.Luma); err != nil {
		return usageError{err}
	}
	if o.Gamma <= 0 {
		return usagef("--gamma must be > 0 (got %v)", o.Gamma)
	}
	if cfg.Input == "-" && cfg.Output == "" {
		return usagef("reading from stdin requires -o")
	}
	if cfg.JSON && cfg.Output == "-" {
		return usagef("--json and -o - both claim stdout; write the file somewhere")
	}
	if cfg.Sidecar && cfg.Output == "-" {
		return usagef("--sidecar needs a file output, not stdout")
	}
	if cfg.Output == "" {
		base := strings.TrimSuffix(cfg.Input, filepath.Ext(cfg.Input))
		cfg.Output = base + ".shr"
	}
	return nil
}

// prodosInfo is the ProDOS metadata block reported in --json.
type prodosInfo struct {
	FileType    uint8  `json:"file_type"`
	FileTypeHex string `json:"file_type_hex"`
	AuxType     uint16 `json:"aux_type"`
	AuxTypeHex  string `json:"aux_type_hex"`
}

// paletteReport is one used palette in --json output.
type paletteReport struct {
	Index  int      `json:"index"`
	Colors []string `json:"colors"` // 16 × "$0RGB"
}

type convertReport struct {
	Input        string           `json:"input"`
	InputFormat  string           `json:"input_format"`
	InputWidth   int              `json:"input_width"`
	InputHeight  int              `json:"input_height"`
	Target       string           `json:"target"`
	Options      pipeline.Options `json:"options"`
	Output       string           `json:"output"`
	OutputSize   int              `json:"output_size"`
	ProDOS       prodosInfo       `json:"prodos"`
	PalettesUsed []paletteReport  `json:"palettes_used"`
	LinePalettes []int            `json:"line_palettes"`
	TimingMS     float64          `json:"timing_ms"`
	Warnings     []string         `json:"warnings"`
}

func runConvert(e *env, cfg convertConfig) error {
	start := time.Now()
	warnings := []string{}
	warnf := func(format string, args ...any) {
		w := fmt.Sprintf(format, args...)
		warnings = append(warnings, w)
		if !cfg.Quiet {
			e.logf("warning: %s", w)
		}
	}
	logv := func(format string, args ...any) {
		if cfg.Verbose {
			e.logf(format, args...)
		}
	}

	// Plumbed-but-inert flags surface as warnings, not silence.
	if cfg.Opt.Seed != 0 {
		warnf("--seed has no effect: no randomized steps in this pipeline")
	}
	if cfg.Opt.Palette != "" {
		return fmt.Errorf("--palette: %w (targets currently use their built-in palettes)", pipeline.ErrNotImplemented)
	}

	tgt, err := pipeline.LookupTarget(cfg.Opt.Target)
	if err != nil {
		return err
	}
	format, err := writer.Lookup(cfg.Opt.Format)
	if err != nil {
		return err
	}

	// Decode.
	var in *os.File
	if cfg.Input == "-" {
		logv("decoding stdin")
	} else {
		logv("decoding %s", cfg.Input)
	}
	reader := e.stdin
	if cfg.Input != "-" {
		if in, err = os.Open(cfg.Input); err != nil {
			return fmt.Errorf("opening input: %w", err)
		}
		defer in.Close()
		reader = in
	}
	img, inFormat, err := source.Decode(reader)
	if err != nil {
		return err
	}
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()
	logv("input: %s %dx%d", inFormat, srcW, srcH)

	// Prepare: linear light → crop → tone → fit/aspect resample.
	li := pix.FromImage(img)
	if c := cfg.Opt.Crop; c != nil {
		if li, err = prepare.Crop(li, c.X, c.Y, c.W, c.H); err != nil {
			return err
		}
		logv("cropped to %dx%d", li.W, li.H)
	}
	adj := prepare.Adjustments{
		Gamma:      cfg.Opt.Gamma,
		Brightness: cfg.Opt.Brightness,
		Contrast:   cfg.Opt.Contrast,
		Saturation: cfg.Opt.Saturation,
	}
	if !adj.Identity() {
		adj.Apply(li)
		logv("applied tone adjustments")
	}
	tw, th := tgt.Geometry()
	par := prepare.PixelAspect(tw, th, cfg.Opt.Aspect)
	if li, err = prepare.Resample(li, tw, th, par, cfg.Opt.Fit); err != nil {
		return err
	}
	logv("resampled to %dx%d (fit=%s, pixel aspect %.2f)", tw, th, cfg.Opt.Fit, par)

	// Convert.
	frame, err := tgt.Convert(li, cfg.Opt)
	if err != nil {
		return fmt.Errorf("target %s: %w", tgt.Name(), err)
	}
	logv("converted with target %s", tgt.Name())

	// Encode container.
	var buf bytes.Buffer
	if err := format.Encode(&buf, frame); err != nil {
		return err
	}

	// Write payload.
	if cfg.Output == "-" {
		if _, err := e.stdout.Write(buf.Bytes()); err != nil {
			return fmt.Errorf("writing to stdout: %w", err)
		}
		logv("wrote %d bytes to stdout", buf.Len())
	} else {
		if err := os.WriteFile(cfg.Output, buf.Bytes(), 0o644); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}
		logv("wrote %s (%d bytes)", cfg.Output, buf.Len())
	}

	// Extras.
	if cfg.PreviewPNG != "" {
		if err := writePreviewPNG(cfg.PreviewPNG, frame); err != nil {
			return err
		}
		logv("wrote preview %s", cfg.PreviewPNG)
	}
	if cfg.Sidecar {
		if err := writer.WriteSidecar(cfg.Output, format, Version); err != nil {
			return err
		}
		logv("wrote sidecar %s", writer.SidecarPath(cfg.Output))
	}

	// Report.
	if cfg.JSON {
		ft, at := format.ProDOS()
		rep := convertReport{
			Input:        cfg.Input,
			InputFormat:  inFormat,
			InputWidth:   srcW,
			InputHeight:  srcH,
			Target:       tgt.Name(),
			Options:      cfg.Opt,
			Output:       cfg.Output,
			OutputSize:   buf.Len(),
			ProDOS:       prodosInfo{ft, fmt.Sprintf("$%02X", ft), at, fmt.Sprintf("$%04X", at)},
			PalettesUsed: usedPalettes(frame),
			LinePalettes: linePalettes(frame),
			TimingMS:     float64(time.Since(start).Microseconds()) / 1000,
			Warnings:     warnings,
		}
		return writeJSON(e.stdout, rep)
	}
	return nil
}

func writePreviewPNG(path string, frame *shr.Frame) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, shr.Render(frame)); err != nil {
		return fmt.Errorf("encoding preview PNG: %w", err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing preview PNG: %w", err)
	}
	return nil
}

// usedPalettes lists each palette referenced by at least one SCB, in index
// order (deterministic).
func usedPalettes(f *shr.Frame) []paletteReport {
	var used [16]bool
	for _, scb := range f.SCB {
		used[shr.SCBPalette(scb)] = true
	}
	out := []paletteReport{}
	for p := 0; p < 16; p++ {
		if !used[p] {
			continue
		}
		colors := make([]string, 16)
		for c := 0; c < 16; c++ {
			colors[c] = f.Palettes[p][c].String()
		}
		out = append(out, paletteReport{Index: p, Colors: colors})
	}
	return out
}

func linePalettes(f *shr.Frame) []int {
	out := make([]int, shr.Height)
	for y, scb := range f.SCB {
		out[y] = int(shr.SCBPalette(scb))
	}
	return out
}
