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

## 2026-11-21 — session 3
- Goal: `agg/stream-parse`.
- Done: `internal/ingest` (Parse, MalformedLog), CLI wiring with the stderr summary, testdata/sample.csv and
  golden/malformed_rows.csv (gate [5] is live). The harness and skeleton commits were made after the owner set the git identity.
- Decisions:
  - Use a line-based `bufio.Reader.ReadSlice` with a manual comma split instead of `encoding/csv`. The data has
    no quotes, `raw_line` needs the original bytes, and this gives 0 allocations per row.
  - `Record.CampaignID` aliases the read buffer, and the aggregator must copy it.
  - Line numbers are physical lines, with the header as line 1. A blank line counts as a malformed row.
  - `malformed_rows.csv` is always written, holding only the header when there are no rejects.
  - Two unchosen rules were added to `ownerPending`: header/empty input is fatal, and spend with more than 2 decimals is malformed.
- Numbers: the real file has 26,843,544 rows, CRLF line endings, 50 campaigns, 0 malformed rows, and spend with ≤ 2 decimals.
  A parse-only run took 3.67s wall time with max RSS 2048 KB. BenchmarkParse runs at 329 MB/s.
- Issues: the Write tool turned the `﻿` escape into a raw BOM, which Go rejects. It now uses `"\xef\xbb\xbf"`.
