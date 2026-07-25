package cli

import "fmt"

const versionUsage = `Print the image2shr version.

Usage:
  image2shr version [flags]
`

func cmdVersion(e *env, args []string) error {
	fs := newFlagSet(e, "version", versionUsage)
	var asJSON bool
	fs.BoolVar(&asJSON, "json", false, "machine-readable version on stdout")
	if err := parse(fs, args); err != nil {
		return err
	}
	if asJSON {
		return writeJSON(e.stdout, struct {
			Tool    string `json:"tool"`
			Version string `json:"version"`
		}{"image2shr", Version})
	}
	fmt.Fprintf(e.stdout, "image2shr %s\n", Version)
	return nil
}
