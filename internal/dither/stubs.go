package dither

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pipeline"
	"github.com/digarok/image2shr/internal/pix"
)

// The remaining dither names from the CLI are registered as stubs: the flag
// value parses and lists today, and Apply fails with a clear "not yet
// implemented" error until the maintainer specs the real algorithms.
func init() {
	for _, name := range []string{
		"atkinson", "jarvis", "sierra", "bayer2", "bayer4", "bayer8",
	} {
		pipeline.RegisterDitherer(stub(name))
	}
}

type stub string

func (s stub) Name() string { return string(s) }

func (s stub) Apply(*pix.LinearImage, pipeline.Plan, pipeline.Options) (pipeline.Indexed, error) {
	return pipeline.Indexed{}, fmt.Errorf("dither %q: %w", string(s), pipeline.ErrNotImplemented)
}
