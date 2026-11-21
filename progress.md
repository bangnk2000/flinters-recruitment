# Progress Log

Append-only, one block per session.

## 2026-11-21 — session 1
- Goal: add a session harness (init-harness, Minimal tier) before writing code.
- Done: AGENTS.md, CLAUDE.md, feature_list.json (7 entries, all todo), init.sh, progress.md,
  session-handoff.md, PROMPTS.md, .gitignore, git init.
- Decisions:
  - Go is the language (owner's choice). It is fast, uses little memory, builds a single binary, and needs only the stdlib.
  - Minimal tier plus a handoff file: the work spans a few sessions, and anything heavier is noise for a challenge repo.
  - Ledger lint uses the jq subset because the repo has no Node.
  - Malformed rows are skipped and counted, and also written to `<output>/malformed_rows.csv` (owner).
- Open: the `ownerPending` items on agg/aggregate, agg/top10-ctr and agg/top10-cpa (tie-break, number
  format, CPA null rendering, CTR when impressions = 0).
