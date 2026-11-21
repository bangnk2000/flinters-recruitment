# Session Handoff Archive
History only, never read on resume. Newest first.

## 2026-11-21 (3) — streaming parser + malformed log (`agg/stream-parse` = pass)
- Done: internal/ingest (Parse, MalformedLog), CLI summary on stderr, sample + golden. Commit: see `git log -1`.
- Proof: `bash init.sh` is green and gate [5] diffs the golden. The real 1GB file took 3.67s with max RSS 2 MB and 0 malformed rows.
- Next: `agg/aggregate`. Replace the no-op `func(ingest.Record) {}` in `internal/cli/cli.go` with the aggregator, and copy
  CampaignID (`m[string(b)]` lookup first, so that 26.8M rows don't allocate).
- Watch out: 2 new `ownerPending` items on stream-parse. When writing Go source, never paste a raw BOM; use `\xef\xbb\xbf`.

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
