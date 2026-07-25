// Command genfixture writes the deterministic test fixture used by the
// golden round-trip tests. Run from the repo root:
//
//	go run ./tools/genfixture
//
// It regenerates internal/cli/testdata/fixture.png. The image is 320x200
// and mixes a smooth horizontal gradient (dither behavior), hard vertical
// bars (edge behavior), a circle (diagonal edges), and saturated color
// patches (luma conversion). No randomness, no timestamps — reruns are
// byte-identical.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	img := image.NewRGBA(image.Rect(0, 0, 320, 200))

	// Background: horizontal grey gradient.
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			v := uint8(x * 255 / 319)
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	// Hard vertical bars, alternating black/white.
	for y := 150; y < 200; y++ {
		for x := 0; x < 320; x++ {
			v := uint8(0)
			if (x/8)%2 == 0 {
				v = 255
			}
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	// Circle centered at (80, 75), radius 40, mid grey.
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			dx, dy := x-80, y-75
			if dx*dx+dy*dy <= 40*40 {
				img.Set(x, y, color.RGBA{128, 128, 128, 255})
			}
		}
	}
	// Saturated color patches (exercise luma weighting).
	patches := []color.RGBA{
		{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {255, 255, 0, 255},
	}
	for i, c := range patches {
		x0 := 180 + (i%2)*50
		y0 := 40 + (i/2)*50
		for y := y0; y < y0+40; y++ {
			for x := x0; x < x0+40; x++ {
				img.Set(x, y, c)
			}
		}
	}

	f, err := os.Create("internal/cli/testdata/fixture.png")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("wrote internal/cli/testdata/fixture.png")
}
