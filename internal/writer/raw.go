package writer

import (
	"fmt"
	"io"

	"github.com/digarok/image2shr/shr"
)

func init() { register(rawFormat{}) }

// rawFormat is the plain 32,768-byte uncompressed screen dump — a byte-exact
// image of IIgs RAM bank $E1, $2000–$9FFF. ProDOS type PIC ($C1),
// auxtype $0000. Layout documented in package shr.
type rawFormat struct{}

func (rawFormat) Name() string        { return "raw" }
func (rawFormat) Description() string { return "uncompressed 32768-byte SHR screen dump" }

func (rawFormat) ProDOS() (uint8, uint16) { return 0xC1, 0x0000 }

func (rawFormat) Encode(w io.Writer, f *shr.Frame) error {
	if _, err := w.Write(f.EncodeRaw()); err != nil {
		return fmt.Errorf("writing raw SHR: %w", err)
	}
	return nil
}
