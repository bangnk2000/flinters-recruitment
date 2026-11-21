#!/usr/bin/env bash
# Verifies the repo from zero. Read-only apart from bin/ and a temp dir. Run from WSL: bash init.sh
set -euo pipefail
cd "$(dirname "$0")"

echo "[0/6] ledger lint (jq subset, init-harness 08 §8.4)"
jq -e '
  (.features | map(.id) | length == (unique | length))
  and ([.features[] | select(.status == "in-progress")] | length <= 1)
  and all(.features[]; .status | IN("todo","in-progress","pass","fail","blocked"))
  and all(.features[] | select(.status == "pass" or .status == "fail"); (.evidence | strings | test("\\S")) // false)
' feature_list.json >/dev/null || { echo "FAIL: ledger lint"; exit 1; }

if [ ! -f go.mod ]; then
  echo "[1-5/6] go gates: PENDING infra/cli-skeleton (no go.mod yet)"
else
  echo "[1/6] gofmt";     unformatted="$(gofmt -l .)"; [ -z "$unformatted" ] || { echo "gofmt needed:"; echo "$unformatted"; exit 1; }
  echo "[2/6] go vet";    go vet ./...
  echo "[3/6] go test";   go test -race ./...
  echo "[4/6] go build";  go build -o bin/aggregator ./cmd/aggregator
  echo "[5/6] runtime proof (testdata/sample.csv vs golden)"
  if [ -f testdata/sample.csv ] && [ -d testdata/golden ]; then
    tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
    ./bin/aggregator --input testdata/sample.csv --output "$tmp"
    for f in testdata/golden/*.csv; do
      diff -u "$f" "$tmp/$(basename "$f")" || { echo "FAIL: $(basename "$f") differs from golden"; exit 1; }
    done
  else
    echo "  PENDING: testdata/sample.csv + testdata/golden/ not created yet"
  fi
fi

echo "[6/6] harness files present"
for f in AGENTS.md CLAUDE.md feature_list.json progress.md session-handoff.md PROMPTS.md; do
  [ -f "$f" ] || { echo "MISSING: $f"; exit 1; }
done
echo "Init OK. Benchmark (not run here): /usr/bin/time -v ./bin/aggregator --input ad_data.csv --output results/"
