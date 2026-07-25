#!/usr/bin/env bash
# permute.sh — run one image through every working flag permutation.
#
# Usage: tools/permute.sh <input-image> [output-dir]
#
# Writes <output-dir>/shr/<name>.shr and <output-dir>/preview/<name>.png,
# where <name> encodes the flags: <dither>_<fit>_<aspect>_<luma>.
# The output-dir defaults to ./permutations.
#
# Permutes only the discrete flags that have working implementations:
# dither none / floyd-steinberg / floyd-steinberg+serpentine, all four
# --fit modes, both --aspect modes, all three --luma weightings (72 total).
# Continuous knobs (--dither-strength, --gamma, --brightness, --contrast,
# --saturation) stay at their defaults, and stub algorithms (atkinson,
# jarvis, sierra, bayer*, packed/apf/brooks formats, scb-mode per-line and
# grouped) are skipped because they exit 1 by design.
#
# Set IMAGE2SHR to use a specific binary; otherwise the repo's bin/image2shr
# is used and built first if missing.
set -euo pipefail

if [[ $# -lt 1 || $# -gt 2 ]]; then
  echo "usage: $0 <input-image> [output-dir]" >&2
  exit 2
fi

input=$1
outdir=${2:-permutations}
repo=$(cd "$(dirname "$0")/.." && pwd)
bin=${IMAGE2SHR:-$repo/bin/image2shr}

[[ -f $input ]] || { echo "input not found: $input" >&2; exit 2; }
if [[ ! -x $bin ]]; then
  echo "building $bin" >&2
  make -C "$repo" build >&2
fi

mkdir -p "$outdir/shr" "$outdir/preview"

# Parallel arrays: extra convert flags and the filename tag for each variant.
dither_flags=("--dither none" "--dither floyd-steinberg" "--dither floyd-steinberg --serpentine")
dither_tags=(none fs fs-serp)
fits=(contain cover stretch none)
aspects=(correct ignore)
lumas=(rec601 rec709 average)

count=0
for i in "${!dither_flags[@]}"; do
  for fit in "${fits[@]}"; do
    for aspect in "${aspects[@]}"; do
      for luma in "${lumas[@]}"; do
        name="${dither_tags[$i]}_${fit}_${aspect}_${luma}"
        # ${dither_flags[$i]} is unquoted on purpose: it must word-split
        # into separate arguments ("--dither floyd-steinberg --serpentine").
        "$bin" convert "$input" -o "$outdir/shr/$name.shr" \
          ${dither_flags[$i]} --fit "$fit" --aspect "$aspect" --luma "$luma"
        "$bin" preview "$outdir/shr/$name.shr" -o "$outdir/preview/$name.png"
        count=$((count + 1))
        echo "[$count] $name" >&2
      done
    done
  done
done

echo "wrote $count .shr files to $outdir/shr with previews in $outdir/preview" >&2
