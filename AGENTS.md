# flinters-recuitment — Agent Instructions

Go CLI for the FV-SEC001 challenge (see `docs/brief.md`, the brief): stream a ~1GB ad-performance CSV,
aggregate per `campaign_id`, and write `top10_ctr.csv` and `top10_cpa.csv`. It is judged on
correct output, clean code, error handling, memory efficiency and tests.

Toolchain: ledger-store=file agent-runtime=claude-code forge=github ci=none
- Harness tier: **Minimal** + `session-handoff.md`. The clean-state checklist lives in this file
  (§Definition of Done), not in a separate file.
- Ledger lint: the `jq` subset of init-harness 08 §8.4 (no Node in this repo). It runs as gate [0] of `init.sh`.
- review gate: N/A · hooks: N/A · orchestration: N/A · deploy: N/A (the CLI is not distributed) · CI_SINCE = false
  (evidence names the gate commands)

## Startup (in order)
1. Read this file completely.
2. Read ONLY the FIRST entry of `session-handoff.md`.
3. `jq '.features[] | {id, status, ownerPending}' feature_list.json` gives the current state. Only ONE feature may be `in-progress`.
4. Run `bash init.sh` (from WSL). If the baseline fails, repair it FIRST.
5. Run `git log --oneline -5`.

## Working Rules
- One feature at a time, picked from `feature_list.json` in dependency order.
- Stay in scope: don't touch files unrelated to the active feature.
- Before calling a feature done, run `bash init.sh`; every gate must be green.
- Go invariants:
  - Use the standard library only, unless this file says why.
  - Read the input as a stream; never load the whole file into memory.
  - Store money as int64 cents, never sum float64.
  - Return errors, don't panic.
  - Exit codes: 0 = ok, 1 = I/O or runtime fault, 2 = usage error.
- Malformed rows (owner decision, 2026-11-21): skip the row and count it. Print the count to stderr.
  Also write every malformed row to `<output>/malformed_rows.csv` with these columns:
  `line_number,reason,raw_line`.
- Owner decisions (2026-11-21, "giữ mặc định hết"):
  - An empty file or a wrong header is fatal (exit 1).
  - Spend with more than 2 decimals is malformed.
  - CTR is 0 when impressions = 0.
  - Ties on CTR or CPA are broken by campaign_id ascending.
  - Format: CTR 4 dp, CPA and spend 2 dp.
  - An empty CPA cell means there are no conversions.
  - The parse stays single-threaded.
  README §Design decisions states these rules.
- Any rule you choose without the owner's decision goes in the entry's `ownerPending`, never only in
  prose.
- `ad_data.csv` (~1GB) is never committed. Tests use `testdata/`. Only `perf/benchmark` reads the real file.
- `PROMPTS.md` holds the owner's prompts **verbatim**. Append new ones as they are typed, and never edit them.

## Definition of Done (and end-of-session clean state)
- [ ] `bash init.sh` is green: ledger lint, gofmt, vet, test -race, build, runtime proof.
- [ ] Runtime proof for the changed behavior: run `bin/aggregator --input testdata/sample.csv --output <tmp>`
      and quote the exit code and the observed output and stderr lines in the evidence.
- [ ] `feature_list.json` reflects reality: status, evidence (with the AC mapping), testedAt.
      `ownerPending` holds every unchosen rule.
- [ ] No stray files. `ad_data.csv` is not staged. `results/` holds the latest real-run output once `perf/benchmark` passes.
- [ ] `progress.md` appended. One `session-handoff.md` entry prepended; keep ≤ 5 entries and move the
      oldest to `docs/harness/session-handoff-archive.md`.
- [ ] `PROMPTS.md` has this session's prompts appended verbatim.
- [ ] Committed with a descriptive message.

## Where Deep Facts Live
- Brief / acceptance criteria: `docs/brief.md`. `README.md` is the submission README (setup, run, numbers, decisions).
- Benchmark: `bash scripts/benchmark.sh` writes `docs/benchmark.log` and `results/`.
- Harness spec (not vendored): `knowledge-system/spec/init-harness/README.md`
- Scaffolding: none. Code layout: `cmd/aggregator/` (main), `internal/…` (packages), `testdata/`.
