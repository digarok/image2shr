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

The canvas is 320x200 when every scanline is in 320 mode. If any scanline
uses 640 mode the canvas must be 640x400: 640-mode pixels need the full
horizontal resolution, and every scanline is doubled vertically to keep the
pixel shape. --size 640 forces the 640x400 canvas for an all-320-mode image
(each pixel becomes a 2x2 block); --size 320 fails if the image contains
640-mode scanlines. --scale enlarges the chosen canvas further.

Supported inputs: uncompressed 32768-byte screens and 38400-byte Brooks
3200-color files, told apart by size. The input "-" reads stdin.

Examples:
  image2shr preview title.shr -o title.png
  image2shr preview title.3200 -o title.png
  image2shr preview title.shr -o big.png --size 640 --scale 2
  image2shr preview title.shr -o - > out.png
`

func cmdPreview(e *env, args []string) error {
	fs := newFlagSet(e, "preview", previewUsage)
	var output, size string
	var scale int
	fs.StringVar(&output, "output", "", "output PNG path, \"-\" for stdout (required)")
	fs.StringVar(&output, "o", "", "alias for --output")
	fs.StringVar(&size, "size", "auto", "canvas size: auto, 320 (320x200), or 640 (640x400)")
	fs.IntVar(&scale, "scale", 1, "integer scale factor for the chosen canvas")

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
	switch size {
	case "auto", "320", "320x200", "640", "640x400":
	default:
		return usagef("--size %q: must be auto, 320 (or 320x200), or 640 (or 640x400)", size)
	}
	if scale < 1 || scale > 16 {
		return usagef("--scale %d out of range 1-16", scale)
	}

	frame, err := readFrame(e, pos[0])
	if err != nil {
		return err
	}

	var img *image.RGBA
	switch size {
	case "320", "320x200":
		if img, err = shr.Render320(frame); err != nil {
			return fmt.Errorf("%s uses 640 mode and requires a 640x400 preview (drop --size 320 or pass --size 640): %w", pos[0], err)
		}
	case "640", "640x400":
		img = shr.Render640(frame)
	default: // auto: 320x200 unless any scanline needs 640 mode
		img = shr.Render(frame)
	}
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

// readFrame loads an uncompressed SHR screen or a Brooks 3200-color file
// from a path or stdin ("-"); the two are distinguished by size.
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
	var frame *shr.Frame
	switch len(data) {
	case shr.FrameSize:
		frame, err = shr.DecodeRaw(data)
	case shr.BrooksSize:
		frame, err = shr.DecodeBrooks(data)
	default:
		err = fmt.Errorf("size %d is neither a raw screen (%d) nor a Brooks 3200 file (%d); compressed/APF reading not supported yet",
			len(data), shr.FrameSize, shr.BrooksSize)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
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
