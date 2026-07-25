package dither

import "github.com/digarok/image2shr/internal/pipeline"

// Classic Floyd–Steinberg error diffusion. Error distribution (× = current
// pixel, weights /16):
//
//	   ×  7
//	3  5  1
func init() {
	pipeline.RegisterDitherer(diffuser{name: "floyd-steinberg", kernel: []tap{
		{1, 0, 7.0 / 16},
		{-1, 1, 3.0 / 16},
		{0, 1, 5.0 / 16},
		{1, 1, 1.0 / 16},
	}})
}
