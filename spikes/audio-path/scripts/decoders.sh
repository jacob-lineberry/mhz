#!/bin/sh
# S1 Q1: compare decoders on a captured live stream.
#   scripts/decoders.sh <capture.mp3> <workdir>
# Capture first, with no ICY metadata so the file is plain MP3:
#   curl -sS -m 1830 -A 'mhz-spike/0' -o cap.mp3 https://stream.nightride.fm/nightride.mp3
set -eu
cap=$1
work=$2
mkdir -p "$work"
go build -o "$work/decbench" ./cmd/decbench
go build -o "$work/pcmcmp" ./cmd/pcmcmp
b=$work/decbench

echo "## clean, full file (CPU, output length)"
"$b" -in "$cap" -pcm "$work/pcm"
echo "## agreement with tphakala (bit-exact minimp3 port) as reference"
for d in hajimehoshi llehouerou; do
	printf '%s: ' "$d"
	"$work/pcmcmp" -ref "$work/pcm/tphakala.s16" -test "$work/pcm/$d.s16"
done
rm -rf "$work/pcm"

head -c 4000000 "$cap" >"$work/head.mp3" # ~100 s at 320 kbps
echo "## join mid-frame: 20 random offsets into the first frame-sized window"
for s in 1 37 111 222 333 417 500 613 700 777 850 901 1000 1044 1100 1203 1300 1337 1400 1440; do
	"$b" -in "$work/head.mp3" -skip $s | tail -3 | sed "s/^/skip=$s /"
done
echo "## flipped bytes"
for f in 1e-6 1e-5 1e-4 1e-3; do
	for seed in 1 2 3; do
		"$b" -in "$work/head.mp3" -flip $f -seed $seed | tail -3 | sed "s/^/flip=$f seed=$seed /"
	done
done
echo "## splices (bytes lost mid-stream, like a reconnect without resync)"
for sp in 400000:1 400000:100 400000:777 400000:3000; do
	"$b" -in "$work/head.mp3" -splice $sp | tail -3 | sed "s/^/splice=$sp /"
done
