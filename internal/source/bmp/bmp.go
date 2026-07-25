// Package bmp implements a Windows BMP decoder for the subset the project
// needs: BITMAPINFOHEADER (or larger V4/V5 headers), BI_RGB (uncompressed)
// only, 8-bit paletted and 24/32-bit truecolor, bottom-up and top-down row
// order, with the mandatory 4-byte row padding.
//
// File structure:
//
//	BITMAPFILEHEADER (14 bytes):
//	  0  2  magic "BM"
//	  2  4  file size (unreliable in the wild; ignored)
//	  6  4  reserved
//	 10  4  offset from start of file to pixel data
//	BITMAPINFOHEADER (40 bytes, larger for V4=108 / V5=124):
//	  0  4  header size
//	  4  4  width  (int32, must be > 0)
//	  8  4  height (int32; > 0 = bottom-up rows, < 0 = top-down)
//	 12  2  planes (must be 1)
//	 14  2  bits per pixel (we accept 8, 24, 32)
//	 16  4  compression (0 = BI_RGB is all we accept)
//	 20 12  image size, resolution (ignored)
//	 32  4  colors used (8-bit: palette entry count; 0 means 256)
//	 36  4  colors important (ignored)
//	then, for 8-bit: palette of B,G,R,X quads
//	then, at the file-header pixel offset: rows, each padded to 4 bytes,
//	pixels stored B,G,R (24-bit) or B,G,R,X (32-bit; X ignored, opaque)
package bmp

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

func init() {
	image.RegisterFormat("bmp", "BM", Decode, DecodeConfig)
}

// maxDim bounds width/height so a corrupt header can't cause a huge allocation.
const maxDim = 1 << 15

type header struct {
	dataOffset uint32
	headerSize uint32
	width      int
	height     int // always positive; topDown records original sign
	topDown    bool
	bpp        int
	clrUsed    uint32
	// bytes consumed from the reader so far (file header + info header)
	consumed int
}

func readHeader(r io.Reader) (*header, error) {
	var fh [14]byte
	if _, err := io.ReadFull(r, fh[:]); err != nil {
		return nil, fmt.Errorf("bmp: reading file header: %w", err)
	}
	if fh[0] != 'B' || fh[1] != 'M' {
		return nil, fmt.Errorf("bmp: bad magic %q, want \"BM\"", fh[0:2])
	}
	h := &header{dataOffset: binary.LittleEndian.Uint32(fh[10:14])}

	var sz [4]byte
	if _, err := io.ReadFull(r, sz[:]); err != nil {
		return nil, fmt.Errorf("bmp: reading info header size: %w", err)
	}
	h.headerSize = binary.LittleEndian.Uint32(sz[:])
	// 40 = BITMAPINFOHEADER; 52/56/108/124 = later variants whose first 40
	// bytes are layout-compatible. The old 12-byte BITMAPCOREHEADER is not.
	if h.headerSize < 40 || h.headerSize > 1024 {
		return nil, fmt.Errorf("bmp: unsupported info header size %d (need BITMAPINFOHEADER, >= 40)", h.headerSize)
	}
	rest := make([]byte, h.headerSize-4)
	if _, err := io.ReadFull(r, rest); err != nil {
		return nil, fmt.Errorf("bmp: reading info header: %w", err)
	}
	h.consumed = 14 + int(h.headerSize)

	width := int32(binary.LittleEndian.Uint32(rest[0:4]))
	height := int32(binary.LittleEndian.Uint32(rest[4:8]))
	planes := binary.LittleEndian.Uint16(rest[8:10])
	h.bpp = int(binary.LittleEndian.Uint16(rest[10:12]))
	compression := binary.LittleEndian.Uint32(rest[12:16])
	h.clrUsed = binary.LittleEndian.Uint32(rest[28:32])

	if compression != 0 {
		return nil, fmt.Errorf("bmp: unsupported compression %d (only BI_RGB is supported)", compression)
	}
	if planes != 1 {
		return nil, fmt.Errorf("bmp: planes = %d, must be 1", planes)
	}
	if h.bpp != 8 && h.bpp != 24 && h.bpp != 32 {
		return nil, fmt.Errorf("bmp: unsupported bit depth %d (supported: 8, 24, 32)", h.bpp)
	}
	if width <= 0 {
		return nil, fmt.Errorf("bmp: invalid width %d", width)
	}
	if height == 0 {
		return nil, fmt.Errorf("bmp: invalid height 0")
	}
	h.width = int(width)
	h.height = int(height)
	if height < 0 { // negative height = top-down row order
		h.topDown = true
		h.height = int(-height)
	}
	if h.width > maxDim || h.height > maxDim {
		return nil, fmt.Errorf("bmp: dimensions %dx%d exceed limit %d", h.width, h.height, maxDim)
	}
	return h, nil
}

// readPalette reads the B,G,R,X color table that follows the info header in
// 8-bit files.
func (h *header) readPalette(r io.Reader) (color.Palette, error) {
	n := int(h.clrUsed)
	if n == 0 {
		n = 256
	}
	if n > 256 {
		return nil, fmt.Errorf("bmp: palette has %d entries, max 256", n)
	}
	quads := make([]byte, 4*n)
	if _, err := io.ReadFull(r, quads); err != nil {
		return nil, fmt.Errorf("bmp: reading palette: %w", err)
	}
	h.consumed += len(quads)
	pal := make(color.Palette, n)
	for i := 0; i < n; i++ {
		pal[i] = color.RGBA{R: quads[4*i+2], G: quads[4*i+1], B: quads[4*i], A: 0xFF}
	}
	return pal, nil
}

// skipToPixels discards any gap between what we've parsed and the pixel data
// offset declared in the file header.
func (h *header) skipToPixels(r io.Reader) error {
	gap := int64(h.dataOffset) - int64(h.consumed)
	if gap < 0 {
		return fmt.Errorf("bmp: pixel data offset %d overlaps headers (%d bytes consumed)", h.dataOffset, h.consumed)
	}
	if gap > 0 {
		if _, err := io.CopyN(io.Discard, r, gap); err != nil {
			return fmt.Errorf("bmp: skipping to pixel data: %w", err)
		}
	}
	return nil
}

// rowStride is the on-disk bytes per row: pixel bytes padded up to a
// multiple of 4.
func (h *header) rowStride() int {
	return (h.width*h.bpp + 31) / 32 * 4
}

// destRow maps an on-disk row index to an image y coordinate: BMP rows are
// stored bottom-up unless the header height was negative.
func (h *header) destRow(i int) int {
	if h.topDown {
		return i
	}
	return h.height - 1 - i
}

// Decode reads a BMP image from r.
func Decode(r io.Reader) (image.Image, error) {
	h, err := readHeader(r)
	if err != nil {
		return nil, err
	}

	var pal color.Palette
	if h.bpp == 8 {
		if pal, err = h.readPalette(r); err != nil {
			return nil, err
		}
	}
	if err := h.skipToPixels(r); err != nil {
		return nil, err
	}

	row := make([]byte, h.rowStride())
	rect := image.Rect(0, 0, h.width, h.height)

	switch h.bpp {
	case 8:
		img := image.NewPaletted(rect, pal)
		for i := 0; i < h.height; i++ {
			if _, err := io.ReadFull(r, row); err != nil {
				return nil, fmt.Errorf("bmp: reading pixel row %d: %w", i, err)
			}
			y := h.destRow(i)
			copy(img.Pix[y*img.Stride:y*img.Stride+h.width], row[:h.width])
		}
		// Guard against indices past the palette (files with clrUsed < 256).
		for _, p := range img.Pix {
			if int(p) >= len(pal) {
				return nil, fmt.Errorf("bmp: pixel index %d out of palette range %d", p, len(pal))
			}
		}
		return img, nil

	case 24, 32:
		bytesPerPixel := h.bpp / 8
		img := image.NewNRGBA(rect)
		for i := 0; i < h.height; i++ {
			if _, err := io.ReadFull(r, row); err != nil {
				return nil, fmt.Errorf("bmp: reading pixel row %d: %w", i, err)
			}
			y := h.destRow(i)
			for x := 0; x < h.width; x++ {
				s := x * bytesPerPixel
				d := y*img.Stride + x*4
				// On disk: B, G, R (, X — ignored, treated as opaque).
				img.Pix[d] = row[s+2]
				img.Pix[d+1] = row[s+1]
				img.Pix[d+2] = row[s]
				img.Pix[d+3] = 0xFF
			}
		}
		return img, nil

	default: // unreachable: bpp validated in readHeader
		return nil, fmt.Errorf("bmp: unsupported bit depth %d", h.bpp)
	}
}

// DecodeConfig returns the dimensions and color model without decoding
// pixel data.
func DecodeConfig(r io.Reader) (image.Config, error) {
	h, err := readHeader(r)
	if err != nil {
		return image.Config{}, err
	}
	cfg := image.Config{Width: h.width, Height: h.height}
	if h.bpp == 8 {
		pal, err := h.readPalette(r)
		if err != nil {
			return image.Config{}, err
		}
		cfg.ColorModel = pal
	} else {
		cfg.ColorModel = color.NRGBAModel
	}
	return cfg, nil
}
