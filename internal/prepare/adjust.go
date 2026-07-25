package prepare

import (
	"math"

	"github.com/digarok/image2shr/internal/pix"
)

// Adjustments are the tone controls from the CLI. Zero-cost defaults
// (gamma=1, brightness=0, contrast=1, saturation=1) leave pixels untouched.
type Adjustments struct {
	Gamma      float64 // power curve, applied as v^(1/gamma); >1 brightens
	Brightness float64 // additive offset in linear light
	Contrast   float64 // scale around linear mid-grey 0.5
	Saturation float64 // 0 = greyscale, 1 = unchanged, >1 = boosted
}

// Identity reports whether applying a would change nothing.
func (a Adjustments) Identity() bool {
	return a.Gamma == 1 && a.Brightness == 0 && a.Contrast == 1 && a.Saturation == 1
}

// saturationLuma is the weight set used for the saturation adjustment
// (independent of the --luma flag, which selects greyscale conversion
// weights for grey targets).
var saturationLuma = pix.LumaWeights{R: 0.2126, G: 0.7152, B: 0.0722} // rec709

// Apply performs the adjustments in place, in linear light, in this order:
// gamma, brightness, contrast, saturation, then a clamp to [0,1].
func (a Adjustments) Apply(img *pix.LinearImage) {
	if a.Identity() {
		return
	}
	invGamma := 1.0
	if a.Gamma > 0 {
		invGamma = 1 / a.Gamma
	}
	for i := 0; i < len(img.Pix); i += 3 {
		r := float64(img.Pix[i])
		g := float64(img.Pix[i+1])
		b := float64(img.Pix[i+2])

		if a.Gamma != 1 && a.Gamma > 0 {
			r = math.Pow(math.Max(r, 0), invGamma)
			g = math.Pow(math.Max(g, 0), invGamma)
			b = math.Pow(math.Max(b, 0), invGamma)
		}
		if a.Brightness != 0 {
			r += a.Brightness
			g += a.Brightness
			b += a.Brightness
		}
		if a.Contrast != 1 {
			r = (r-0.5)*a.Contrast + 0.5
			g = (g-0.5)*a.Contrast + 0.5
			b = (b-0.5)*a.Contrast + 0.5
		}
		if a.Saturation != 1 {
			y := float64(saturationLuma.Apply(float32(r), float32(g), float32(b)))
			r = y + (r-y)*a.Saturation
			g = y + (g-y)*a.Saturation
			b = y + (b-y)*a.Saturation
		}
		img.Pix[i] = pix.Clamp01(float32(r))
		img.Pix[i+1] = pix.Clamp01(float32(g))
		img.Pix[i+2] = pix.Clamp01(float32(b))
	}
}

// PixelAspect returns the destination pixel aspect ratio (height/width in
// display units) for an SHR framebuffer of the given size on the 4:3
// screen: 320×200 → 1.2, 640×200 → 2.4. aspect is the --aspect flag value;
// "ignore" returns 1.0.
func PixelAspect(w, h int, aspect string) float64 {
	if aspect == "ignore" {
		return 1.0
	}
	return (float64(w) / float64(h)) * 3.0 / 4.0
}
