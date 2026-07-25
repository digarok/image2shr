// Package source decodes input images. PNG, JPEG, and GIF come from the
// standard library; BMP is our own decoder in the bmp subpackage. All
// register themselves with the image package, so one Decode entry point
// dispatches by sniffing the file's magic bytes.
package source

import (
	"fmt"
	"image"
	"io"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "github.com/digarok/image2shr/internal/source/bmp"
)

// Decode reads an image from r, sniffing the format. It returns the decoded
// image and the format name ("png", "jpeg", "gif", "bmp").
func Decode(r io.Reader) (image.Image, string, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		return nil, "", fmt.Errorf("decoding image: %w (supported formats: png, jpeg, gif, bmp)", err)
	}
	return img, format, nil
}
