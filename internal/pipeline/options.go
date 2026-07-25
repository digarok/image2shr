// Package pipeline defines the seams of the conversion pipeline: the
// Options bag, the Plan and Indexed intermediate types, the Target /
// Ditherer / PalettePlanner interfaces, their registries, and the shared
// plan→dither→pack glue.
package pipeline

import "errors"

// ErrNotImplemented marks a registered-but-stubbed algorithm. Stubs exist so
// flag values parse today and real algorithms can drop in later.
var ErrNotImplemented = errors.New("not yet implemented")

// CropRect is the --crop X,Y,W,H rectangle in source-image coordinates.
type CropRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// Options carries every resolved convert flag. The CLI validates flag values
// before anything runs; algorithm code can trust the names it receives exist
// in the relevant registry.
type Options struct {
	Target         string    `json:"target"`
	Format         string    `json:"format"`
	Dither         string    `json:"dither"`
	DitherStrength float64   `json:"dither_strength"`
	Serpentine     bool      `json:"serpentine"`
	SCBMode        string    `json:"scb_mode"`
	Palette        string    `json:"palette,omitempty"`
	Fit            string    `json:"fit"`
	Aspect         string    `json:"aspect"`
	Crop           *CropRect `json:"crop,omitempty"`
	Gamma          float64   `json:"gamma"`
	Brightness     float64   `json:"brightness"`
	Contrast       float64   `json:"contrast"`
	Saturation     float64   `json:"saturation"`
	Luma           string    `json:"luma"`
	Seed           int64     `json:"seed"`
}

// DefaultOptions returns the documented flag defaults.
func DefaultOptions() Options {
	return Options{
		Target:         "shr320-grey16",
		Format:         "auto", // raw, or brooks for 3200-color frames
		Dither:         "floyd-steinberg",
		DitherStrength: 1.0,
		SCBMode:        "auto", // resolves to each planner's natural mode
		Fit:            "contain",
		Aspect:         "correct",
		Gamma:          1.0,
		Brightness:     0.0,
		Contrast:       1.0,
		Saturation:     1.0,
		Luma:           "rec601",
	}
}
