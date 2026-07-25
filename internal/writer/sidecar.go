package writer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Sidecar is the <output>.meta.json document written by --sidecar: ProDOS
// metadata for downstream disk tools (Cadius, CiderPress). Field order is
// fixed by the struct, so output is deterministic.
type Sidecar struct {
	File        string `json:"file"`          // output file name (base name)
	Format      string `json:"format"`        // container name, e.g. "raw"
	FileType    uint8  `json:"file_type"`     // ProDOS file type, decimal
	FileTypeHex string `json:"file_type_hex"` // e.g. "$C1"
	AuxType     uint16 `json:"aux_type"`
	AuxTypeHex  string `json:"aux_type_hex"` // e.g. "$0000"
	Tool        string `json:"tool"`         // "image2shr"
	Version     string `json:"version"`
}

// NewSidecar builds the sidecar document for an output written in format f.
func NewSidecar(outputPath string, f Format, version string) Sidecar {
	ft, at := f.ProDOS()
	return Sidecar{
		File:        filepath.Base(outputPath),
		Format:      f.Name(),
		FileType:    ft,
		FileTypeHex: fmt.Sprintf("$%02X", ft),
		AuxType:     at,
		AuxTypeHex:  fmt.Sprintf("$%04X", at),
		Tool:        "image2shr",
		Version:     version,
	}
}

// SidecarPath returns the sidecar file path for an output path:
// <output>.meta.json.
func SidecarPath(outputPath string) string { return outputPath + ".meta.json" }

// WriteSidecar writes the sidecar JSON next to the output file.
func WriteSidecar(outputPath string, f Format, version string) error {
	doc := NewSidecar(outputPath, f, version)
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding sidecar: %w", err)
	}
	data = append(data, '\n')
	path := SidecarPath(outputPath)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing sidecar %s: %w", path, err)
	}
	return nil
}
