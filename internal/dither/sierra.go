package dither

import "github.com/digarok/image2shr/internal/pipeline"

// Sierra (three-row) error diffusion. Error distribution (× = current
// pixel, weights /32):
//
//	      ×  5  3
//	2  4  5  4  2
//	   2  3  2
//
// Similar spread to Jarvis–Judice–Ninke with a lighter kernel.
func init() {
	pipeline.RegisterDitherer(diffuser{name: "sierra", kernel: []tap{
		{1, 0, 5.0 / 32}, {2, 0, 3.0 / 32},
		{-2, 1, 2.0 / 32}, {-1, 1, 4.0 / 32}, {0, 1, 5.0 / 32}, {1, 1, 4.0 / 32}, {2, 1, 2.0 / 32},
		{-1, 2, 2.0 / 32}, {0, 2, 3.0 / 32}, {1, 2, 2.0 / 32},
	}})
}
