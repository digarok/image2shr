# image2shr — working notes

Command-line converter from modern images to Apple IIgs Super Hi-Res.
The authoritative project brief is `docs/SPEC.md`; the design overview is
`docs/ARCHITECTURE.md`. Read both before changing anything structural.

## Build / test

    make build          # → bin/image2shr (version via -ldflags from git describe)
    make test           # go test ./...
    make lint           # gofmt -l check + go vet
    make cross          # dist/ binaries: darwin/linux/windows × amd64/arm64

Golden files: `go test ./internal/cli -run Golden -update` regenerates
`internal/cli/testdata/golden.*` after an intentional output change. Never
regenerate to make a failing test pass without understanding why it changed.

## Hard rules

- **Zero third-party dependencies.** The `require` block in go.mod stays empty.
  Ask the maintainer before ever adding one.
- **stdout is payload-only** (binary output or the --json object). All logs,
  warnings, progress → stderr. Exit codes: 0 success, 1 runtime failure,
  2 usage error.
- **Determinism:** same input + flags + version ⇒ byte-identical output. No map
  iteration order in output paths, no unseeded randomness, no timestamps in
  output files. Registries iterate sorted by name.
- **No panics outside main.** Errors wrapped with context.
- Hardware byte layouts (SCB bits, $0RGB color encoding, 640-mode
  pixel-position→palette-group mapping) are heavily commented where
  implemented — keep it that way. The 640 mapping lives in exactly one place:
  `shr.PaletteIndex640`.

## Layout

- `shr/` — the only public package: Frame model, packers, renderer, raw codec.
- `internal/pix` — linear-light float image + sRGB/luma conversions.
- `internal/source` — decode dispatch; `internal/source/bmp` is our own decoder.
- `internal/prepare` — crop/fit/aspect/adjustments.
- `internal/pipeline` — Options, Plan, Indexed, the Target/Ditherer/PalettePlanner
  interfaces, registries, Run glue.
- `internal/planner`, `internal/dither`, `internal/target` — implementations;
  one file per algorithm/target, registered in init().
- `internal/writer` — output containers + ProDOS metadata + sidecar.
- `internal/cli` — subcommands and flag parsing (stdlib `flag` only).

## Conventions

- Algorithms are deliberately the dumbest correct versions until the maintainer
  specs the real ones — do not "improve" them unasked; do not build multi-SCB /
  multi-palette logic unasked.
- Table-driven tests; tiny handmade fixtures for binary formats.
- Commit in logical increments with clear messages.
- When the brief is ambiguous, ask instead of guessing.
