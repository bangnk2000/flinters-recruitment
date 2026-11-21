#!/usr/bin/env bash
# Benchmarks bin/aggregator on the real dataset. Writes the result files to
# results/ and the measurements to docs/benchmark.log.
# Usage: bash scripts/benchmark.sh [input.csv]   (RUNS=n to change the run count)
set -euo pipefail
cd "$(dirname "$0")/.."

input="${1:-ad_data.csv}"
runs="${RUNS:-3}"
log=docs/benchmark.log
[ -f "$input" ] || { echo "missing $input (unzip ad_data.csv.zip first)" >&2; exit 1; }
command -v /usr/bin/time >/dev/null || { echo "needs GNU time (/usr/bin/time)" >&2; exit 1; }

go build -o bin/aggregator ./cmd/aggregator
mkdir -p docs results

{
  echo "# aggregator benchmark, $(date -u +%Y-%m-%dT%H:%M:%SZ), commit $(git rev-parse --short HEAD)"
  echo "input : $input, $(stat -c %s "$input") bytes"
  echo "cpu   : $(lscpu | sed -n 's/^Model name: *//p'), $(nproc) logical CPUs available"
  echo "memory: $(free -h | awk '/^Mem:/{print $2}') total"
  echo "os    : $(uname -sr)"
  echo "go    : $(go version | cut -d' ' -f3-)"
  echo "note  : page cache is warm after the first read; dropping it needs root, so runs are warm-cache"
  echo
  /usr/bin/time -f "read only (cat > /dev/null): wall=%e s" cat "$input" > /dev/null
  echo
  ref="$(mktemp -d)"
  /usr/bin/time -f "single-thread reference (--workers 1): wall=%e s user=%U s sys=%S s maxrss=%M kB" \
    ./bin/aggregator --input "$input" --output "$ref" --workers 1
  echo
  for i in $(seq 1 "$runs"); do
    /usr/bin/time -f "run $i (--workers $(nproc)): wall=%e s user=%U s sys=%S s maxrss=%M kB" \
      ./bin/aggregator --input "$input" --output results
  done
  echo
  diff -r -q "$ref" results && echo "parallel outputs are byte-identical to the single-thread reference"
  rm -rf "$ref"
} 2>&1 | tee "$log"
