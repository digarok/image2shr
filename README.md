# image2shr

Convert modern images (PNG, JPEG, GIF, BMP) into Apple IIgs Super Hi-Res
files — from the command line, with no dependencies, deterministically.

Built for two consumers:

- **Pipelines** — Makefiles and shell scripts in IIgs game/demo development.
  Quiet by default, useful exit codes, byte-identical output for identical
  inputs and flags.
- **UI front-ends** — a GUI can shell out to this binary: stdout carries only
  the payload (binary or `--json`), everything else goes to stderr, and
  `preview` renders any SHR file to PNG exactly as the IIgs would display it.

Written in Go, standard library only. Licensed GPL-3.0.

## Build

```sh
make build          # → bin/image2shr
make test           # run all tests
make lint           # gofmt + go vet
make cross          # dist/ binaries for darwin/linux/windows × amd64/arm64
```

## Usage

```sh
# Convert with defaults (target shr320-grey16, Floyd–Steinberg dither):
image2shr convert photo.png                    # writes photo.shr

# Everything explicit:
image2shr convert -o title.shr -t shr320-grey16 \
    --dither floyd-steinberg --dither-strength 0.8 --serpentine \
    --fit cover --aspect correct --luma rec709 title.jpg

# See what the IIgs will display, before ever leaving your desk:
image2shr preview title.shr -o check.png
image2shr preview --scale 2 title.shr -o big.png

# Or get the preview in the same run:
image2shr convert --preview-png check.png -o title.shr title.jpg

# Inspect any raw SHR file:
image2shr inspect title.shr
image2shr inspect --json title.shr

# List conversion targets:
image2shr targets

# Fully piped:
cat in.png | image2shr convert -o - - > out.shr

# Machine-readable result for tooling (payload goes to the file):
image2shr convert --json -o out.shr in.png

# ProDOS metadata for Cadius / CiderPress:
image2shr convert --sidecar -o out.shr in.png   # also writes out.shr.meta.json
```

Run `image2shr <command> --help` for every flag with examples.

### Exit codes

| Code | Meaning |
|------|---------|
| 0    | success |
| 1    | runtime failure (bad input file, unimplemented algorithm, I/O error) |
| 2    | usage error (unknown flag/value, missing argument) |

### Machine interface guarantees

- stdout is payload-only: the converted bytes (`-o -`), or the `--json`
  object. Logs, warnings, and progress go to stderr.
- Same input + same flags + same version ⇒ byte-identical output.
- `--json` reports input dimensions, resolved options, output size, ProDOS
  file type/auxtype, palettes used, per-line palette assignments, timing,
  and a warnings array.

## Output formats

| `--format` | Contents | ProDOS type/aux | Status |
|------------|----------|-----------------|--------|
| `raw`      | 32,768-byte uncompressed screen dump | PIC `$C1`/`$0000` | ✔ |
| `packed`   | PackBytes-compressed screen | PNT `$C0`/`$0001` | stub |
| `apf`      | Apple Preferred Format (block-structured) | PNT `$C0`/`$0002` | stub |
| `brooks`   | Brooks 3200-color | PIC `$C1`/`$0002` | stub |

## Targets

| Target | Geometry | Description |
|--------|----------|-------------|
| `shr320-grey16` | 320×200 | greyscale, single 16-grey palette, all SCBs `$00` |

More targets (color, multi-palette, 640 mode, 3200-color) plug into the same
pipeline; see `docs/ARCHITECTURE.md` for where.

## Documentation

- `docs/SPEC.md` — the project brief and SHR hardware reference (byte
  layouts, SCB bits, the 640-mode pixel-position palette mapping).
- `docs/ARCHITECTURE.md` — pipeline stages and extension points.

## License

GPL-3.0 — see `LICENSE`.
