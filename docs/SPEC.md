# image2shr — Project Brief (SPEC)

> This is the original kickoff brief, kept verbatim as the project reference.

We are building a new command-line image conversion tool that converts modern
images into Apple IIgs Super Hi-Res (SHR) formats.

## Project identity

Working name: **image2shr** (single binary, `image2shr` on the command line).
Language: Go (latest stable). Module path `github.com/digarok/image2shr`.
**Zero third-party dependencies.** `go.mod` must have an empty require block and
stay that way. Standard library only. If you ever think you need a dependency,
stop and ask me first. PNG/JPEG/GIF decoding comes from `image/png`,
`image/jpeg`, `image/gif`. BMP we write ourselves (see below).

License: **GPL3**

## What this tool is for

Two consumers, both first-class:

1. **Pipelines** — invoked from Makefiles and shell scripts during Apple IIgs
   game/demo/app development. Deterministic, scriptable, quiet by default,
   useful exit codes.
2. **UI front-ends** — a GUI wraps this binary and calls it as a backend for
   live preview and final export. That means machine-readable output, fast
   startup, and the ability to emit a preview PNG of exactly what the IIgs will
   display.

Explicit anti-goal: do not imitate the CLI conventions of b2d / bmp2dhr (terse
positional mystery arguments like `bmp2dhr foo.bmp d p1 a`). We want a modern,
discoverable, subcommand-and-long-flag CLI. Everything configurable, nothing
cryptic.

## Hardware background you must encode correctly

The Super Hi-Res screen is a fixed 32,768-byte structure (a raw dump of IIgs
RAM bank $E1, $2000–$9FFF). File offsets in an uncompressed `.shr` /
PIC $C1/aux $0000 file:

| Offset        | Size  | Contents                                  |
|---------------|-------|-------------------------------------------|
| $0000–$7CFF   | 32000 | Pixel data, 200 lines × 160 bytes         |
| $7D00–$7DC7   | 200   | Scanline Control Bytes (SCB), one per line|
| $7DC8–$7DFF   | 56    | Reserved, must be zero                    |
| $7E00–$7FFF   | 512   | 16 palettes × 16 colors × 2 bytes         |

SCB byte layout (one per scanline, so mode is per-line):

    bit 7   : horizontal resolution — 0 = 320, 1 = 640
    bit 6   : scanline interrupt enable
    bit 5   : color fill mode (320 mode only)
    bit 4   : reserved, must be 0
    bits 0-3: palette number, 0–15

Color entries are 12-bit RGB444 stored little-endian as $0RGB: low byte = GB
(high nibble green, low nibble blue), high byte = 0R (high nibble must be zero,
low nibble red). So $0F80 is stored as bytes $80 $0F.

**320 mode:** one byte = two pixels, high nibble is the left pixel. The 4-bit
value indexes directly into that line's 16-color palette.

**640 mode:** one byte = four pixels, 2 bits each, bits 7–6 leftmost.
Critically, the 2-bit value does not index the palette directly — the pixel's
position within the byte selects which group of four palette entries it can
reach:

    pixel position in byte:  0     1     2     3
    palette entries usable:  8-11  12-15 0-3   4-7

So `paletteIndex = []int{8, 12, 0, 4}[x % 4] + value`. Any 640-mode writer must
take the absolute x coordinate, not just a color value. Get this wrong and
everything looks plausible but is subtly incorrect — put it behind one
well-tested function and a table-driven unit test.

**Color fill mode (320 only):** when SCB bit 5 is set, pixel value 0 means
"repeat the previous pixel's color" rather than "palette entry 0". Undefined if
the leftmost pixel of the line is 0. Support it in the model and the renderer,
default it off.

**Pixel aspect ratio:** SHR displays on a 4:3 screen, so pixels are not square.
In 320 mode a pixel is roughly 1.2× taller than wide; in 640 mode roughly 2.4×
taller than wide. Aspect handling must be an explicit, configurable step, not
an accident of resizing.

## The central data structure

Everything in the program converges on one type. Build it first.

```go
// package shr

type RGB12 struct{ R, G, B uint8 } // each 0-15

type Frame struct {
    Pixels   [32000]byte   // raw packed pixel bytes, 200 lines × 160
    SCB      [200]byte
    Palettes [16][16]RGB12
}
```

Conversion produces a Frame. Writers serialize a Frame. The renderer turns a
Frame back into an RGB image. This keeps encoders, algorithms, and file formats
fully decoupled.

## Pipeline architecture

Model the conversion as discrete, individually testable stages. Each stage is
an interface so I can drop in real algorithms later:

1. **Decode** — source file → image.Image (PNG/JPEG/GIF stdlib, BMP ours).
2. **Prepare** — crop, fit/scale, aspect correction, gamma / brightness /
   contrast / saturation. Work in a linear-light float RGB buffer, not 8-bit
   sRGB, so error diffusion behaves.
3. **Plan palettes** — a PalettePlanner decides the 16 palettes and which
   palette each scanline uses (the "multi-SCB" problem). Output:
   `[16][16]RGB12` + `[200]uint8` assignments.
4. **Quantize + dither** — a Ditherer maps prepared pixels to palette indices,
   respecting the per-line palette and (in 640 mode) the pixel-position
   constraint.
5. **Pack** — indices → packed bytes + SCB table → Frame.
6. **Write** — Frame → output container.

Suggested interfaces (adjust if you have a better shape, but keep the seams):

```go
type PalettePlanner interface {
    Name() string
    Plan(src *LinearImage, opt Options) (Plan, error)
}

type Ditherer interface {
    Name() string
    Apply(src *LinearImage, plan Plan, opt Options) (Indexed, error)
}

type Target interface {
    Name() string           // e.g. "shr320-grey16"
    Description() string
    Geometry() (w, h int)
    Convert(src *LinearImage, opt Options) (*shr.Frame, error)
}
```

Targets live in a registry keyed by name so `image2shr targets` can list them
and new ones are one file each.

## Output formats

v1 must write:

- **raw** — the plain 32,768-byte screen dump. ProDOS type PIC ($C1),
  auxtype $0000.

Design the writer interface so these can be added later without refactoring —
leave stub files with the format documented but unimplemented, returning a
clear "not yet implemented" error:

- PackBytes-compressed SHR — PNT ($C0) / $0001
- Apple Preferred Format (block-structured: MAIN, MULTIPAL, …) — PNT ($C0) / $0002
- Brooks 3200-color (32000 pixel bytes + 200 per-line palettes, palettes stored
  in reverse color order, no SCB) — PIC ($C1) / $0002

Also emit ProDOS type/auxtype metadata so downstream tools (Cadius,
CiderPress) can set it: a `--sidecar` flag writing `<output>.meta.json`, plus
the values in `--json` output.

## CLI design

Subcommand style. Long flags with occasional short aliases. `--help` on every
subcommand with real examples.

    image2shr convert [flags] <input>     # input "-" reads stdin
    image2shr preview <file.shr> -o out.png   # render an SHR file exactly as the IIgs would display it
    image2shr inspect <file.shr>          # report modes used, palettes, SCB histogram, unique colors
    image2shr targets                     # list conversion targets
    image2shr version

convert flags (starting set — implement the plumbing for all of them, even
where the underlying algorithm is a stub):

    -o, --output PATH          output file, "-" for stdout
    -t, --target NAME          conversion target (default shr320-grey16)
        --format NAME          container: auto (default — raw, or brooks for
                               3200-color frames), raw, packed, apf, brooks
        --dither NAME          none, floyd-steinberg, atkinson, jarvis, sierra, bayer2/4/8
        --dither-strength F    0.0-1.0 (default 1.0)
        --serpentine           serpentine scan for error diffusion
        --scb-mode NAME        auto, single, banded, grouped, per-line
                               (auto = the target's natural mode; multi-palette
                               modes need a multi-palette target)
        --palette NAME|FILE    named builtin or a palette file
        --fit MODE             contain (default), cover, stretch, none
        --aspect MODE          correct (default), ignore
        --crop X,Y,W,H
        --gamma F  --brightness F  --contrast F  --saturation F
        --luma NAME            rec601, rec709, average — for greyscale targets
        --preview-png PATH     also write a round-tripped PNG of the result
        --sidecar              write ProDOS type/auxtype metadata JSON
        --seed N               make any randomized step deterministic
        --json                 machine-readable result on stdout
    -v, --verbose  -q, --quiet

Hard rules for the machine interface:

- stdout carries only the payload — binary output, or the `--json` object.
  Every log, warning, and progress message goes to stderr. A GUI must be able
  to trust stdout.
- Exit codes: 0 success, 1 runtime failure, 2 usage error.
- Same input + same flags + same version = byte-identical output. No map
  iteration order, no unseeded randomness, no timestamps in output.
- `--json` reports: input dimensions, target, resolved options, output size,
  ProDOS type/auxtype, palettes used, per-line palette assignments, timing,
  warnings array.

Standard library only — implement flag parsing with `flag`, or a thin
subcommand wrapper around it if `flag` alone gets awkward. Do not hand-roll a
clever parser.

## First milestone — build exactly this and stop

Target: **shr320-grey16**. 320×200, single palette (palette 0) for all 200
lines, 16 evenly spaced greys. Palette entries 0–15 are $0000, $0111, $0222, …
$0FFF, i.e. `RGB12{n,n,n}`. All SCBs = $00.

Deliverables:

1. Repo skeleton, go.mod with no dependencies, Makefile with build, test,
   lint, cross (darwin/linux/windows × amd64/arm64).
2. Own BMP decoder in internal/source/bmp — BITMAPINFOHEADER, BI_RGB, 8/24/32-bit,
   bottom-up and top-down, correct row padding. Table-driven tests with tiny
   handmade fixtures. Register it alongside the stdlib decoders behind one
   `source.Decode(io.Reader)`.
3. The shr.Frame type, packers for both 320 and 640 (yes, write the 640 packer
   now — just don't expose a 640 target yet), SCB helpers, palette
   serialization.
4. The renderer: Frame → image.RGBA, honoring per-line mode, per-line palette,
   640 pixel-position mapping, and fill mode. This is the correctness anchor
   for the whole project.
5. convert, preview, inspect, targets, version subcommands wired up with the
   full flag set above.
6. The shr320-grey16 target end to end, with the simplest correct
   implementations: nearest-luma quantization, and Floyd–Steinberg as the one
   working ditherer. Register `none` too. Other dither names should parse and
   return a clear "not implemented" error.
7. Tests: unit tests per package, plus golden-file round-trip tests
   (fixture.png → convert → .shr → preview → compare to golden PNG) so I can
   refactor algorithms later without fear. Byte-exact comparison on the .shr.
8. README.md with usage examples, and docs/ARCHITECTURE.md describing the
   pipeline stages and where a new target/ditherer/planner plugs in. Keep a
   CLAUDE.md with build/test commands and conventions.

Do not invent sophisticated conversion algorithms. Do not build multi-palette
or multi-SCB logic yet, do not write a k-means quantizer, do not guess at
640-mode dithering strategies. I will describe those to you in detail once the
framework and harness are solid. Where an algorithm belongs, define the
interface and provide the dumbest correct implementation, clearly marked.

## Working style

- Go idioms: small packages, interfaces defined where consumed, errors wrapped
  with context, no panics outside main. gofmt and go vet clean.
- Comment the hardware-specific bits heavily with the byte layouts above —
  future me will need them, and they are not guessable from the code.
- Commit in logical increments with clear messages.
- When something in this brief is ambiguous or you think it is wrong, ask
  instead of guessing.
