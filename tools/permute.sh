#!/usr/bin/env bash
# permute.sh — run one image through every working flag permutation.
#
# Usage: tools/permute.sh <input-image> [output-dir]
#
# Writes <output-dir>/shr/<name>.shr and <output-dir>/preview/<name>.png,
# where <name> encodes the flags: <target>_<dither>_<fit>_<aspect>_<luma>,
# with the swept --scb-mode folded into the target tag for color256
# (color256-banded, color256-grouped, color256-perline).
# The output-dir defaults to ./permutations.
#
# Permutes only the discrete flags that have working implementations:
# all three targets (grey16, color16, color256), dither none /
# floyd-steinberg / floyd-steinberg+serpentine, all four --fit modes, both
# --aspect modes, all three --luma weightings for grey16 only (the color
# targets ignore --luma), and the three multi-palette --scb-mode strategies
# for color256 only ("single" there would duplicate the color16 output).
# 168 outputs total.
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
targets=(shr320-grey16 shr320-color16 shr320-color256)
fits=(contain cover stretch none)
aspects=(correct ignore)

count=0
for target in "${targets[@]}"; do
  # --luma only affects the greyscale reduction; the color targets ignore
  # it, so sweeping it there would just write identical files under three
  # names. --scb-mode only branches inside color256 ("auto" elsewhere).
  case $target in
  shr320-grey16) lumas=(rec601 rec709 average); scbs=(auto) ;;
  shr320-color256) lumas=(rec601); scbs=(banded grouped per-line) ;;
  *) lumas=(rec601); scbs=(auto) ;;
  esac
  for scb in "${scbs[@]}"; do
    ttag=${target#shr320-}
    [[ $scb != auto ]] && ttag="${ttag}-${scb//-/}"
    for i in "${!dither_flags[@]}"; do
      for fit in "${fits[@]}"; do
        for aspect in "${aspects[@]}"; do
          for luma in "${lumas[@]}"; do
            name="${ttag}_${dither_tags[$i]}_${fit}_${aspect}_${luma}"
            # ${dither_flags[$i]} is unquoted on purpose: it must word-split
            # into separate arguments ("--dither floyd-steinberg --serpentine").
            "$bin" convert "$input" -t "$target" -o "$outdir/shr/$name.shr" \
              ${dither_flags[$i]} --scb-mode "$scb" \
              --fit "$fit" --aspect "$aspect" --luma "$luma"
            "$bin" preview "$outdir/shr/$name.shr" -o "$outdir/preview/$name.png"
            count=$((count + 1))
            echo "[$count] $name" >&2
          done
        done
      done
    done
  done
done

echo "wrote $count .shr files to $outdir/shr with previews in $outdir/preview" >&2
