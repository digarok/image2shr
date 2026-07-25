package writer

import (
	"fmt"
	"io"

	"github.com/digarok/image2shr/shr"
)

func init() { register(brooksFormat{}) }

// brooksFormat is the Brooks 3200-color format (".3200"). ProDOS type PIC
// ($C1), auxtype $0002. The byte layout — 32,000 pixel bytes followed by
// 200 per-line palettes with colors in reverse entry order, 38,400 bytes
// total — is documented and implemented in shr.EncodeBrooks.
//
// Any frame can be written: a 3200-color frame stores its per-line
// palettes directly, and a normal 16-palette frame is expanded by copying
// each line's SCB-selected palette into that line's slot.
type brooksFormat struct{}

func (brooksFormat) Name() string { return "brooks" }
func (brooksFormat) Description() string {
	return "Brooks 3200-color: per-line palettes, reverse color order"
}

func (brooksFormat) ProDOS() (uint8, uint16) { return 0xC1, 0x0002 }

func (brooksFormat) Encode(w io.Writer, f *shr.Frame) error {
	buf, err := f.EncodeBrooks()
	if err != nil {
		return fmt.Errorf("format \"brooks\": %w", err)
	}
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("format \"brooks\": %w", err)
	}
	return nil
}
