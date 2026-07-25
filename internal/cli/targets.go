package cli

import (
	"fmt"

	"github.com/digarok/image2shr/internal/pipeline"
)

const targetsUsage = `List the available conversion targets.

Usage:
  image2shr targets [flags]

Examples:
  image2shr targets
  image2shr targets --json
`

type targetReport struct {
	Name        string `json:"name"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Description string `json:"description"`
}

func cmdTargets(e *env, args []string) error {
	fs := newFlagSet(e, "targets", targetsUsage)
	var asJSON bool
	fs.BoolVar(&asJSON, "json", false, "machine-readable list on stdout")
	pos, err := parse(fs, args)
	if err != nil {
		return err
	}
	if len(pos) != 0 {
		return usagef("targets takes no arguments")
	}

	list := pipeline.Targets() // sorted by name
	if asJSON {
		out := make([]targetReport, len(list))
		for i, t := range list {
			w, h := t.Geometry()
			out[i] = targetReport{Name: t.Name(), Width: w, Height: h, Description: t.Description()}
		}
		return writeJSON(e.stdout, out)
	}
	for _, t := range list {
		w, h := t.Geometry()
		fmt.Fprintf(e.stdout, "%-16s %dx%d  %s\n", t.Name(), w, h, t.Description())
	}
	return nil
}
