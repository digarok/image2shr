package cli

import (
	"fmt"

	"github.com/digarok/image2shr/shr"
)

const inspectUsage = `Report an SHR file's modes, palettes, SCB usage, and color counts.

Usage:
  image2shr inspect [flags] <file.shr>

Only uncompressed 32768-byte screens are supported so far. The input "-"
reads stdin.

Examples:
  image2shr inspect title.shr
  image2shr inspect --json title.shr
`

// inspectReport is the machine-readable form of the inspect output.
type inspectReport struct {
	File         string          `json:"file"`
	Size         int             `json:"size"`
	Lines320     int             `json:"lines_320"`
	Lines640     int             `json:"lines_640"`
	FillLines    int             `json:"fill_lines"`
	IRQLines     int             `json:"interrupt_lines"`
	SCBHistogram []scbBucket     `json:"scb_histogram"` // palettes actually referenced
	PalettesUsed []paletteReport `json:"palettes_used"`
	LinePalettes []int           `json:"line_palettes"`
	UniqueColors int             `json:"unique_colors"`
}

type scbBucket struct {
	Palette int `json:"palette"`
	Lines   int `json:"lines"`
}

func cmdInspect(e *env, args []string) error {
	fs := newFlagSet(e, "inspect", inspectUsage)
	var asJSON bool
	fs.BoolVar(&asJSON, "json", false, "machine-readable report on stdout")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 1 {
		return usagef("inspect needs exactly one .shr file; see --help")
	}

	frame, err := readFrame(e, pos[0])
	if err != nil {
		return err
	}
	rep := inspectFrame(pos[0], frame)

	if asJSON {
		return writeJSON(e.stdout, rep)
	}

	// Human-readable report. This IS the payload of inspect, so it goes to
	// stdout.
	w := e.stdout
	fmt.Fprintf(w, "file:            %s (%d bytes)\n", rep.File, rep.Size)
	fmt.Fprintf(w, "scanlines:       %d in 320 mode, %d in 640 mode\n", rep.Lines320, rep.Lines640)
	fmt.Fprintf(w, "fill lines:      %d\n", rep.FillLines)
	fmt.Fprintf(w, "interrupt lines: %d\n", rep.IRQLines)
	fmt.Fprintf(w, "SCB histogram:\n")
	for _, b := range rep.SCBHistogram {
		fmt.Fprintf(w, "  palette %2d: %3d lines\n", b.Palette, b.Lines)
	}
	fmt.Fprintf(w, "unique colors:   %d\n", rep.UniqueColors)
	for _, p := range rep.PalettesUsed {
		fmt.Fprintf(w, "palette %d:\n ", p.Index)
		for _, c := range p.Colors {
			fmt.Fprintf(w, " %s", c)
		}
		fmt.Fprintln(w)
	}
	return nil
}

// inspectFrame computes the report. Unique colors counts distinct RGB12
// values among palette entries actually referenced by pixels (resolving the
// 640-mode position mapping through shr.PaletteIndex640).
func inspectFrame(name string, f *shr.Frame) inspectReport {
	rep := inspectReport{
		File:         name,
		Size:         shr.FrameSize,
		PalettesUsed: usedPalettes(f),
		LinePalettes: linePalettes(f),
	}

	var scbCount [16]int
	colorSet := map[shr.RGB12]bool{}
	for y := 0; y < shr.Height; y++ {
		scb := f.SCB[y]
		pn := shr.SCBPalette(scb)
		scbCount[pn]++
		pal := &f.Palettes[pn]
		if shr.SCBIsFill(scb) {
			rep.FillLines++
		}
		if shr.SCBIsInterrupt(scb) {
			rep.IRQLines++
		}
		row := f.Line(y)
		if shr.SCBIs640(scb) {
			rep.Lines640++
			vals := shr.UnpackLine640(row)
			for x, v := range vals {
				colorSet[pal[shr.PaletteIndex640(x, v)]] = true
			}
		} else {
			rep.Lines320++
			idx := shr.UnpackLine320(row)
			fill := shr.SCBIsFill(scb)
			for x, v := range idx {
				if fill && v == 0 && x > 0 {
					continue // repeats the previous color; adds nothing new
				}
				colorSet[pal[v]] = true
			}
		}
	}
	for p, n := range scbCount {
		if n > 0 {
			rep.SCBHistogram = append(rep.SCBHistogram, scbBucket{Palette: p, Lines: n})
		}
	}
	rep.UniqueColors = len(colorSet)
	return rep
}
