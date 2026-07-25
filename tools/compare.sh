#!/usr/bin/env bash
# compare.sh — run one image through every conversion mode, nothing else.
#
# Usage: tools/compare.sh <input-image> [output-dir]
#
# The narrow companion to permute.sh: every target and color256 --scb-mode
# strategy, each with and without dithering, all at the default framing
# (--fit contain, --aspect correct) so the outputs differ only in how they
# use the palette hardware. 12 outputs total. The color3200 .shr files are
# Brooks-format (38,400 bytes) despite the extension; preview sniffs by size.
#
# Writes <output-dir>/shr/<name>.shr and <output-dir>/preview/<name>.png,
# where <name> is <mode>_<dither> (e.g. color256-grouped_fs).
# The output-dir defaults to ./compare.
#
# Set IMAGE2SHR to use a specific binary; otherwise the repo's bin/image2shr
# is used and built first if missing.
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: $0 <input-image> [output-dir]" >&2
  exit 2
fi

input=$1
outdir=${2:-compare}
repo=$(cd "$(dirname "$0")/.." && pwd)
bin=${IMAGE2SHR:-$repo/bin/image2shr}

[[ -f $input ]] || { echo "input not found: $input" >&2; exit 2; }
if [[ ! -x $bin ]]; then
  echo "building $bin" >&2
  make -C "$repo" build >&2
fi

mkdir -p "$outdir/shr" "$outdir/preview"

# Parallel arrays: target + scb-mode per conversion mode, and its name tag.
mode_targets=(shr320-grey16 shr320-color16 shr320-color256 shr320-color256 shr320-color256 shr320-color3200)
mode_scbs=(auto auto banded grouped per-line auto)
mode_tags=(grey16 color16 color256-banded color256-grouped color256-perline color3200)

count=0
for m in "${!mode_targets[@]}"; do
  for dither in none floyd-steinberg; do
    dtag=fs
    [[ $dither == none ]] && dtag=none
    name="${mode_tags[$m]}_${dtag}"
    "$bin" convert "$input" -t "${mode_targets[$m]}" --scb-mode "${mode_scbs[$m]}" \
      --dither "$dither" -o "$outdir/shr/$name.shr"
    "$bin" preview "$outdir/shr/$name.shr" -o "$outdir/preview/$name.png"
    count=$((count + 1))
    echo "[$count] $name" >&2
  done
done

echo "wrote $count .shr files to $outdir/shr with previews in $outdir/preview" >&2
