package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// writeJSON emits one indented JSON object to w (stdout for --json
// payloads). Struct field order fixes the output; nothing here may iterate
// a map.
func writeJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}
	return nil
}
