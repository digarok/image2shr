package writer

import (
	"fmt"
	"io"

	"github.com/digarok/image2shr/shr"
)

func init() { register(packedFormat{}) }

// packedFormat will be the PackBytes-compressed SHR screen. ProDOS type
// PNT ($C0), auxtype $0001.
//
// Format (once implemented): the 32,768-byte raw screen compressed with
// Apple's PackBytes RLE scheme (as in the IIgs Toolbox Misc Tools PackBytes
// call). PackBytes emits runs of the form:
//
//	flag byte: top 2 bits = operation, bottom 6 bits = count-1
//	  00 cccccc : 1–64 literal bytes follow
//	  01 cccccc : one byte follows, repeated (count+1) × ... (3, 5, 6, 7 reps
//	              encodings per the Toolbox reference)
//	  10 cccccc : four bytes follow, repeated count+1 times
//	  11 cccccc : one byte follows, repeated 4×(count+1) times
//
// STUB: returns ErrNotImplemented until the compressor is written.
type packedFormat struct{}

func (packedFormat) Name() string        { return "packed" }
func (packedFormat) Description() string { return "PackBytes-compressed SHR screen (stub)" }

func (packedFormat) ProDOS() (uint8, uint16) { return 0xC0, 0x0001 }

func (packedFormat) Encode(io.Writer, *shr.Frame) error {
	return fmt.Errorf("format \"packed\": %w", ErrNotImplemented)
}
