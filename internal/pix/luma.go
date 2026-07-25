package pix

import "fmt"

// LumaWeights are the per-channel weights used to reduce RGB to a single
// luma value. Applied to linear-light values.
type LumaWeights struct{ R, G, B float32 }

// Luma weight sets selectable with --luma. Names match the CLI flag values.
var lumaWeights = map[string]LumaWeights{
	"rec601":  {0.299, 0.587, 0.114},
	"rec709":  {0.2126, 0.7152, 0.0722},
	"average": {1.0 / 3, 1.0 / 3, 1.0 / 3},
}

// LumaNames lists the valid --luma flag values, sorted.
var LumaNames = []string{"average", "rec601", "rec709"}

// LumaOf resolves a --luma flag value to its weights.
func LumaOf(name string) (LumaWeights, error) {
	w, ok := lumaWeights[name]
	if !ok {
		return LumaWeights{}, fmt.Errorf("unknown luma %q (valid: average, rec601, rec709)", name)
	}
	return w, nil
}

// Apply reduces one linear RGB value to luma.
func (w LumaWeights) Apply(r, g, b float32) float32 {
	return w.R*r + w.G*g + w.B*b
}

// ToGrey returns a copy of src with every pixel replaced by its luma
// (r = g = b = Y), so greyscale targets can reuse generic RGB machinery.
func ToGrey(src *LinearImage, w LumaWeights) *LinearImage {
	out := New(src.W, src.H)
	for i := 0; i < len(src.Pix); i += 3 {
		y := w.Apply(src.Pix[i], src.Pix[i+1], src.Pix[i+2])
		out.Pix[i], out.Pix[i+1], out.Pix[i+2] = y, y, y
	}
	return out
}
