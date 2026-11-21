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

## 2026-11-21 — session 2
- Goal: `infra/cli-skeleton`.
- Done: go.mod (module `adagg`, go 1.25), `cmd/aggregator/main.go` (it only calls `cli.Run`),
  `internal/cli` with flag parsing, input/output validation, exit codes 0/1/2, and a table test (9 cases).
  infra/cli-skeleton = pass (evidence in the ledger).
- Decisions:
  - `main` stays a one-liner so that every behavior is testable through `cli.Run(args, stdout, stderr)`.
  - `-h` goes to stdout with exit 0. Usage errors go to stderr with exit 2 and the usage text attached.
  - The output dir is created with MkdirAll.
  - The module is named `adagg`, which gives short import paths and no forge URL yet.
- Issues: there is no git identity in WSL or Windows, so the session-1 and session-2 commits are not made yet.
