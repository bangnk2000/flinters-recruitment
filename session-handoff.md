# Session Handoff
Newest first. On resume read ONLY the first entry. Close: prepend one entry, keep ≤ 5,
move the oldest verbatim to docs/harness/session-handoff-archive.md.

## 2026-11-21 (2) — CLI skeleton done (`infra/cli-skeleton` = pass)
- Done: go.mod, cmd/aggregator, internal/cli (Run + tests). NOT committed: git has no author identity yet.
- Proof: `bash init.sh` is green (gate [5] PENDING). The built binary was run against each exit path; the observed
  lines are quoted in the ledger evidence.
- Next: set the git identity and commit the harness + skeleton → `agg/stream-parse`. Create testdata/sample.csv
  with malformed rows and testdata/golden/ in that feature so that gate [5] goes live.
- Watch out: WSL's default shell is fish, so run scripts as `bash file.sh`. `internal/cli.run` only opens and stats the
  input; stream-parse plugs the reader in there.

## 2026-11-21 (1) — harness bootstrapped, no code yet (`infra/cli-skeleton` = todo)
- Done: harness files + git init. No Go code exists yet.
- Proof: `bash init.sh`. The ledger lint passes; the go gates print PENDING until go.mod exists.
- Next: `infra/cli-skeleton` → `agg/stream-parse`. Create testdata/sample.csv (include malformed rows)
  and testdata/golden/ early so that gate [5] goes live.
- Watch out: never commit ad_data.csv (1GB). The ownerPending items need the owner's answer before
  agg/top10-* can pass. Using the defaults is fine, but keep the items listed.
