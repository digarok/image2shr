package writer

import (
	"fmt"
	"io"

	"github.com/digarok/image2shr/shr"
)

func init() { register(apfFormat{}) }

// apfFormat will be Apple Preferred Format, the block-structured picture
// container used by IIgs paint programs. ProDOS type PNT ($C0),
// auxtype $0002.
//
// Format (once implemented): a sequence of named blocks, each
//
//	4 bytes  block length (little-endian, includes the length field itself)
//	pString  block name (length-prefixed ASCII), e.g. "MAIN", "MULTIPAL",
//	         "PATS", "SCIB", "MASK", "NOTE"
//	...      block body
//
// The MAIN block holds: master SCB mode word, number of color tables, the
// color tables, number of scanlines, per-scanline SCB words, and the pixel
// data as PackBytes-compressed scanline runs. MULTIPAL (for 3200-color
// images) holds one 16-color table per scanline. Unknown blocks must be
// preserved by readers.
//
// STUB: returns ErrNotImplemented until the block writer is built.
type apfFormat struct{}

func (apfFormat) Name() string        { return "apf" }
func (apfFormat) Description() string { return "Apple Preferred Format, block-structured (stub)" }

func (apfFormat) ProDOS() (uint8, uint16) { return 0xC0, 0x0002 }

func (apfFormat) Encode(io.Writer, *shr.Frame) error {
	return fmt.Errorf("format \"apf\": %w", ErrNotImplemented)
}
