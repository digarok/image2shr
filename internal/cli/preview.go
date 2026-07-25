package cli

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"

	"github.com/digarok/image2shr/shr"
)

const previewUsage = `Render an SHR file to PNG exactly as the IIgs would display it.

Usage:
  image2shr preview [flags] <file.shr>

Only uncompressed 32768-byte screens are supported so far. The canvas is
640x200 (320-mode pixels are two canvas pixels wide); SHR pixels are not
square, so pass --scale to enlarge for viewing. The input "-" reads stdin.

Examples:
  image2shr preview title.shr -o title.png
  image2shr preview --scale 2 title.shr -o big.png
  image2shr preview title.shr -o - > out.png
`

func cmdPreview(e *env, args []string) error {
	fs := newFlagSet(e, "preview", previewUsage)
	var output string
	var scale int
	fs.StringVar(&output, "output", "", "output PNG path, \"-\" for stdout (required)")
	fs.StringVar(&output, "o", "", "alias for --output")
	fs.IntVar(&scale, "scale", 1, "integer scale factor for the 640x200 canvas")

	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usagef("preview needs exactly one .shr file; see --help")
	}
	if output == "" {
		return usagef("preview requires -o / --output")
	}
	if scale < 1 || scale > 16 {
		return usagef("--scale %d out of range 1-16", scale)
	}

	frame, err := readFrame(e, pos[0])
	if err != nil {
		return err
	}

	img := shr.Render(frame)
	out := image.Image(img)
	if scale > 1 {
		out = scaleNearest(img, scale)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return fmt.Errorf("encoding PNG: %w", err)
	}
	if output == "-" {
		_, err = e.stdout.Write(buf.Bytes())
		return err
	}
	if err := os.WriteFile(output, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing PNG: %w", err)
	}
	return nil
}

// readFrame loads an uncompressed SHR screen from a path or stdin ("-").
func readFrame(e *env, path string) (*shr.Frame, error) {
	var data []byte
	var err error
	if path == "-" {
		if data, err = io.ReadAll(e.stdin); err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
	} else {
		if data, err = os.ReadFile(path); err != nil {
			return nil, fmt.Errorf("reading input: %w", err)
		}
	}
	frame, err := shr.DecodeRaw(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w (compressed/APF/Brooks reading not supported yet)", path, err)
	}
	return frame, nil
}

// scaleNearest integer-scales with nearest neighbor — previews must stay
// pixel-exact, no filtering.
func scaleNearest(src *image.RGBA, s int) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx()*s, b.Dy()*s))
	for y := 0; y < b.Dy()*s; y++ {
		for x := 0; x < b.Dx()*s; x++ {
			dst.Set(x, y, src.At(b.Min.X+x/s, b.Min.Y+y/s))
		}
	}
	return dst
}
