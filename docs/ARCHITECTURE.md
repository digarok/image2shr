# Architecture

The whole program converges on one type: `shr.Frame`, a complete Super
Hi-Res screen (pixel bytes + SCBs + palettes). Conversion produces a Frame,
writers serialize a Frame, the renderer turns a Frame back into RGB. That
keeps algorithms, encoders, and file formats fully decoupled.

## Pipeline

```
input file
   │  internal/source          Decode: sniff png/jpeg/gif (stdlib) or bmp (ours)
   ▼
image.Image
   │  internal/pix             one-time sRGB → linear-light float conversion
   ▼
pix.LinearImage
   │  internal/prepare         Prepare: crop → gamma/brightness/contrast/
   │                           saturation → fit/aspect resample to target size
   ▼
pix.LinearImage (target geometry)
   │  Target.Convert           (internal/target, one file per target)
   │    │
   │    ├─ PalettePlanner.Plan   → pipeline.Plan     (palettes + per-line assignment)
   │    ├─ Ditherer.Apply        → pipeline.Indexed  (raw pixel values)
   │    └─ pipeline.PackFrame    → shr.Frame         (pack bytes + SCBs + palettes)
   ▼
shr.Frame
   │  internal/writer          Format.Encode: raw today; packed/apf/brooks stubs
   ▼
output file (+ optional preview PNG via shr.Render, + optional .meta.json sidecar)
```

Every stage boundary is a plain data type (`LinearImage`, `Plan`, `Indexed`,
`Frame`), so each stage tests in isolation and algorithms swap without
touching neighbors.

## Package map

| Package | Role |
|---------|------|
| `shr` (public) | Hardware model: `Frame`, `RGB12`, SCB helpers, 320/640 packers/unpackers, raw codec, renderer. No image processing, no options. |
| `internal/pix` | Linear-light float buffer, sRGB transfer functions, luma weights. |
| `internal/source` | Decode dispatch; `source/bmp` is our own BMP decoder. |
| `internal/prepare` | Crop, aspect-aware box resampler, tone adjustments. |
| `internal/pipeline` | The seams: `Options`, `Plan`, `Indexed`, the three interfaces, registries, `Run`/`PackFrame` glue. |
| `internal/planner` | `PalettePlanner` implementations: `Fixed`, `AdaptiveColor16`, `AdaptiveColor256` (multi-SCB: banded / adaptive-grouped / per-line strategies via `--scb-mode`), `AdaptiveColor3200` (one palette per scanline, Brooks). |
| `internal/dither` | `Ditherer` implementations (v1: `none`, `floyd-steinberg`; stubs for the rest). |
| `internal/target` | `Target` implementations (v1: `shr320-grey16`). |
| `internal/writer` | Output containers + ProDOS type/auxtype + sidecar. |
| `internal/cli` | Subcommands, flag parsing, stdout/stderr discipline, exit codes, `--json`. |

## Hardware invariants (enforced in `shr`, tested there)

- Raw file layout: 32,000 pixel bytes, 200 SCBs at `$7D00`, 56 reserved zero
  bytes, 512 palette bytes at `$7E00`.
- Colors are 12-bit `$0RGB`, stored little-endian (`GB`, `0R`).
- 320 mode: high nibble = left pixel, direct palette index.
- 640 mode: the 2-bit pixel value is NOT a palette index. The pixel's
  position within its byte picks the palette group: positions 0,1,2,3 reach
  entries 8–11, 12–15, 0–3, 4–7. This mapping lives in exactly one function,
  `shr.PaletteIndex640`, used by the packing docs, the renderer, and
  inspect. Never reimplement it.
- Color fill mode (320 only, SCB bit 5): pixel value 0 repeats the previous
  color. The renderer honors it; no v1 target emits it.
- `shr.Render` is the correctness anchor: per-line mode/palette/fill honored,
  640-mode pixels resolved through `PaletteIndex640`. The canvas is 320×200
  when every scanline is in 320 mode; any 640-mode scanline forces 640×400
  (scanlines doubled vertically, 320-mode pixels doubled horizontally).
  `Render320`/`Render640` force a canvas — `Render320` errors on frames with
  640-mode lines. Golden tests compare against this output.

## Extension points

**New target** — one file in `internal/target/`. Implement
`pipeline.Target`, call `pipeline.RegisterTarget` in `init()`. Compose a
planner + `pipeline.Run` (as `shr320grey16.go` does), or do something
entirely custom that returns a `*shr.Frame`.

**New ditherer** — one file in `internal/dither/`. Implement
`pipeline.Ditherer` over a `LinearImage` + `Plan`, register in `init()`; the
CLI flag value is the `Name()`. Replace the corresponding stub in
`stubs.go`. 640-mode dithering must choose 2-bit values whose meaning
depends on x (via `shr.PaletteIndex640`) — currently rejected as
not-implemented in `check320`.

**New palette planner** — `internal/planner/`. Emit per-line palette
assignments in `Plan.Line` and palettes in `Plan.Palettes`;
`pipeline.PackFrame` already writes per-line SCBs. `AdaptiveColor256` is the
reference multi-SCB planner — it routes the `--scb-mode` strategies (banded,
grouped with DP-placed cuts, fully dynamic per-line clustering).

**New output format** — one file in `internal/writer/`. Implement `Format`
(including `ProDOS()`), register in `init()`. The `packed`, `apf`, and
`brooks` stubs document their on-disk layouts in comments.

## Determinism rules

- No map iteration in any output path — registries list sorted by name.
- No timestamps in output files; `--json` timing is the only time-derived
  value anywhere, and it's in the report, not the artifact.
- Any future randomized algorithm must draw from `Options.Seed`.
- Golden tests (`internal/cli/golden_test.go`) pin the whole pipeline:
  fixture.png → byte-exact `.shr` → pixel-exact preview PNG. Regenerate
  with `go test ./internal/cli -run Golden -update` only for intentional
  output changes. The fixture is generated by `tools/genfixture`.
