package pix

import "math"

// LinearToOklab converts linear-light sRGB components (the LinearImage
// representation, 0..1) to Oklab coordinates.
//
// Oklab (Björn Ottosson, 2020; public-domain reference implementation) is a
// perceptually uniform opponent space: Euclidean distance approximates how
// different two colors LOOK, unlike distance in RGB where dark colors are
// crowded together and green differences are undervalued. Palette planners
// cluster in this space so "least error" means least perceived error.
//
// L is perceived lightness (~0..1); a and b are the green–red and
// blue–yellow opponent axes centered on 0.
func LinearToOklab(r, g, b float64) (L, A, B float64) {
	// Linear sRGB → LMS-like cone response.
	l := 0.4122214708*r + 0.5363325363*g + 0.0514459929*b
	m := 0.2119034982*r + 0.6806995451*g + 0.1073969566*b
	s := 0.0883024619*r + 0.2817188376*g + 0.6299787005*b

	// Cube root is Oklab's perceptual compression.
	lc, mc, sc := math.Cbrt(l), math.Cbrt(m), math.Cbrt(s)

	L = 0.2104542553*lc + 0.7936177850*mc - 0.0040720468*sc
	A = 1.9779984951*lc - 2.4285922050*mc + 0.4505937099*sc
	B = 0.0259040371*lc + 0.7827717662*mc - 0.8086757660*sc
	return L, A, B
}
