# Session Handoff
Newest first. On resume read ONLY the first entry. Close: prepend one entry, keep ≤ 5,
move the oldest verbatim to docs/harness/session-handoff-archive.md.

## 2026-11-21 (1) — harness bootstrapped, no code yet (`infra/cli-skeleton` = todo)
- Done: harness files + git init. No Go code exists yet.
- Proof: `bash init.sh`. The ledger lint passes; the go gates print PENDING until go.mod exists.
- Next: `infra/cli-skeleton` → `agg/stream-parse`. Create testdata/sample.csv (include malformed rows)
  and testdata/golden/ early so that gate [5] goes live.
- Watch out: never commit ad_data.csv (1GB). The ownerPending items need the owner's answer before
  agg/top10-* can pass. Using the defaults is fine, but keep the items listed.
