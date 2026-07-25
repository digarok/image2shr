package writer

import (
	"fmt"
	"io"

	"github.com/digarok/image2shr/shr"
)

func init() { register(brooksFormat{}) }

// brooksFormat will be the Brooks 3200-color format. ProDOS type PIC ($C1),
// auxtype $0002.
//
// Format (once implemented), total 38,400 bytes:
//
//	$0000–$7CFF  32000 bytes  pixel data, 200 lines × 160 bytes (as raw SHR)
//	$7D00–$95FF   6400 bytes  200 palettes, one 32-byte palette per line,
//	                          colors stored in REVERSE order: entry 15 first,
//	                          entry 0 last. No SCB table — every line
//	                          implicitly uses its own palette, 320 mode.
//
// Producing real 3200-color output also needs a per-line palette planner;
// this writer is only the container.
//
// STUB: returns ErrNotImplemented until then.
type brooksFormat struct{}

func (brooksFormat) Name() string { return "brooks" }
func (brooksFormat) Description() string {
	return "Brooks 3200-color: per-line palettes, reverse color order (stub)"
}

func (brooksFormat) ProDOS() (uint8, uint16) { return 0xC1, 0x0002 }

func (brooksFormat) Encode(io.Writer, *shr.Frame) error {
	return fmt.Errorf("format \"brooks\": %w", ErrNotImplemented)
}
