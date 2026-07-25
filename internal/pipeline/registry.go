package pipeline

import (
	"fmt"
	"sort"
)

// Registries for targets, ditherers, and planners. Implementations register
// in init(); lookups are by flag value. Listing is always sorted by name so
// no output path depends on map iteration order.

var (
	targets   = map[string]Target{}
	ditherers = map[string]Ditherer{}
	planners  = map[string]PalettePlanner{}
)

func RegisterTarget(t Target) { targets[t.Name()] = t }

func LookupTarget(name string) (Target, error) {
	t, ok := targets[name]
	if !ok {
		return nil, fmt.Errorf("unknown target %q (run \"image2shr targets\" for the list)", name)
	}
	return t, nil
}

// Targets returns all registered targets sorted by name.
func Targets() []Target {
	names := make([]string, 0, len(targets))
	for n := range targets {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Target, len(names))
	for i, n := range names {
		out[i] = targets[n]
	}
	return out
}

func RegisterDitherer(d Ditherer) { ditherers[d.Name()] = d }

func LookupDitherer(name string) (Ditherer, error) {
	d, ok := ditherers[name]
	if !ok {
		return nil, fmt.Errorf("unknown dither %q (valid: %s)", name, DithererNames())
	}
	return d, nil
}

// DithererNames returns the registered ditherer names sorted, joined with ", ".
func DithererNames() string {
	names := make([]string, 0, len(ditherers))
	for n := range ditherers {
		names = append(names, n)
	}
	sort.Strings(names)
	s := ""
	for i, n := range names {
		if i > 0 {
			s += ", "
		}
		s += n
	}
	return s
}

func RegisterPlanner(p PalettePlanner) { planners[p.Name()] = p }

func LookupPlanner(name string) (PalettePlanner, error) {
	p, ok := planners[name]
	if !ok {
		return nil, fmt.Errorf("unknown planner %q", name)
	}
	return p, nil
}
