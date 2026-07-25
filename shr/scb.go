package shr

// Scanline Control Byte (SCB) layout — one per scanline, so the mode is
// per-line:
//
//	bit 7   : horizontal resolution — 0 = 320, 1 = 640
//	bit 6   : scanline interrupt enable
//	bit 5   : color fill mode (320 mode only)
//	bit 4   : reserved, must be 0
//	bits 0-3: palette number, 0–15
const (
	SCBMode640     = 0x80
	SCBInterrupt   = 0x40
	SCBFill        = 0x20 // 320 mode only
	SCBPaletteMask = 0x0F
)

// MakeSCB assembles a scanline control byte. The palette number is masked
// to 0–15; the reserved bit 4 is always zero.
func MakeSCB(mode640, interrupt, fill bool, palette uint8) byte {
	b := palette & SCBPaletteMask
	if mode640 {
		b |= SCBMode640
	}
	if interrupt {
		b |= SCBInterrupt
	}
	if fill {
		b |= SCBFill
	}
	return b
}

// SCBIs640 reports whether the SCB selects 640 mode (bit 7).
func SCBIs640(scb byte) bool { return scb&SCBMode640 != 0 }

// SCBIsFill reports whether the SCB enables color fill mode (bit 5,
// meaningful in 320 mode only).
func SCBIsFill(scb byte) bool { return scb&SCBFill != 0 }

// SCBIsInterrupt reports whether the SCB enables the scanline interrupt (bit 6).
func SCBIsInterrupt(scb byte) bool { return scb&SCBInterrupt != 0 }

// SCBPalette returns the palette number 0–15 (bits 0–3).
func SCBPalette(scb byte) uint8 { return scb & SCBPaletteMask }

// LineIs640 reports whether scanline y is in 640 mode.
func (f *Frame) LineIs640(y int) bool { return SCBIs640(f.SCB[y]) }

// LineIsFill reports whether scanline y has color fill mode enabled.
func (f *Frame) LineIsFill(y int) bool { return SCBIsFill(f.SCB[y]) }
