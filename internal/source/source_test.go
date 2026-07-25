package source

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestDecodePNG(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{0xFF, 0, 0, 0xFF})
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	img, format, err := Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if format != "png" {
		t.Errorf("format = %q, want png", format)
	}
	if img.Bounds().Dx() != 2 {
		t.Errorf("width = %d, want 2", img.Bounds().Dx())
	}
}

func TestDecodeBMPDispatch(t *testing.T) {
	// Minimal 1×1 24-bit BMP, hand-assembled: the point is that image.Decode
	// dispatches "BM" files to our registered decoder.
	data := []byte{
		'B', 'M', 58, 0, 0, 0, 0, 0, 0, 0, 54, 0, 0, 0, // file header
		40, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 24, 0, // info header...
		0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0x00, 0x00, 0xFF, 0x00, // one red pixel + pad
	}
	img, format, err := Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if format != "bmp" {
		t.Errorf("format = %q, want bmp", format)
	}
	got := color.RGBAModel.Convert(img.At(0, 0)).(color.RGBA)
	if got != (color.RGBA{0xFF, 0, 0, 0xFF}) {
		t.Errorf("pixel = %v, want red", got)
	}
}

func TestDecodeUnknownFormat(t *testing.T) {
	if _, _, err := Decode(bytes.NewReader([]byte("not an image at all"))); err == nil {
		t.Error("expected error for unknown format")
	}
}
