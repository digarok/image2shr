// Package writer serializes a shr.Frame into an output container and knows
// each container's ProDOS file type / auxtype so downstream tools (Cadius,
// CiderPress) can set them.
package writer

import (
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/digarok/image2shr/shr"
)

// ErrNotImplemented marks a documented-but-stubbed output format.
var ErrNotImplemented = errors.New("not yet implemented")

// Format is one output container. Implementations register in init().
type Format interface {
	Name() string // flag value: raw, packed, apf, brooks
	Description() string
	// ProDOS returns the file type and auxtype for this container,
	// e.g. $C1/$0000 for a raw screen dump.
	ProDOS() (fileType uint8, auxType uint16)
	Encode(w io.Writer, f *shr.Frame) error
}

var formats = map[string]Format{}

func register(f Format) { formats[f.Name()] = f }

// Lookup resolves a --format flag value.
func Lookup(name string) (Format, error) {
	f, ok := formats[name]
	if !ok {
		return nil, fmt.Errorf("unknown format %q (valid: %s)", name, names())
	}
	return f, nil
}

func names() string {
	ns := make([]string, 0, len(formats))
	for n := range formats {
		ns = append(ns, n)
	}
	sort.Strings(ns)
	out := ""
	for i, n := range ns {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}

// Formats returns all registered formats sorted by name.
func Formats() []Format {
	ns := make([]string, 0, len(formats))
	for n := range formats {
		ns = append(ns, n)
	}
	sort.Strings(ns)
	out := make([]Format, len(ns))
	for i, n := range ns {
		out[i] = formats[n]
	}
	return out
}
