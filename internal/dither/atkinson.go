package dither

import "github.com/digarok/image2shr/internal/pipeline"

// Atkinson dithering (Bill Atkinson, original Macintosh). Error
// distribution (× = current pixel, weights /8):
//
//	   ×  1  1
//	1  1  1
//	   1
//
// The weights sum to 6/8: a quarter of the error is deliberately dropped,
// trading exact level preservation for higher local contrast and less
// "worm" texture — the classic Mac look.
func init() {
	pipeline.RegisterDitherer(diffuser{name: "atkinson", kernel: []tap{
		{1, 0, 1.0 / 8},
		{2, 0, 1.0 / 8},
		{-1, 1, 1.0 / 8},
		{0, 1, 1.0 / 8},
		{1, 1, 1.0 / 8},
		{0, 2, 1.0 / 8},
	}})
}
