package dither

import "github.com/digarok/image2shr/internal/pipeline"

// Jarvis–Judice–Ninke error diffusion. Error distribution (× = current
// pixel, weights /48):
//
//	      ×  7  5
//	3  5  7  5  3
//	1  3  5  3  1
//
// Wider than Floyd–Steinberg, so the error spreads more smoothly at the
// cost of some sharpness.
func init() {
	pipeline.RegisterDitherer(diffuser{name: "jarvis", kernel: []tap{
		{1, 0, 7.0 / 48}, {2, 0, 5.0 / 48},
		{-2, 1, 3.0 / 48}, {-1, 1, 5.0 / 48}, {0, 1, 7.0 / 48}, {1, 1, 5.0 / 48}, {2, 1, 3.0 / 48},
		{-2, 2, 1.0 / 48}, {-1, 2, 3.0 / 48}, {0, 2, 5.0 / 48}, {1, 2, 3.0 / 48}, {2, 2, 1.0 / 48},
	}})
}
