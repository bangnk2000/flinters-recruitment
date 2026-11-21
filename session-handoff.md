# Session Handoff
Newest first. On resume read ONLY the first entry. Close: prepend one entry, keep ≤ 5,
move the oldest verbatim to docs/harness/session-handoff-archive.md.

## 2026-11-21 (9) — parallel parse, 3.8× faster (`perf/parallel-parse` = pass)
- Done: ingest pipeline (chunk reader → N workers → ordered collector), agg.Merge, `--workers`, README/benchmark updated.
  NOT pushed: the owner decides when to push.
- Proof: init.sh is green (goldens with 1 and nproc workers). TestParseWorkersAgree passes and -race is clean. Real file: 1.04s against 3.96s, RSS ~11 MB,
  and outputs are byte-identical.
- Next: nothing in the ledger. The owner may push (`git push`); the remote is bangnk2000/flinters-recruitment.
- Watch out: the owner removed the repo-operation prompts from PROMPTS.md and force-pushed, so keep only development prompts there.
  Any parse change must keep TestParseWorkersAgree green.

## 2026-11-21 (8) — submission docs; every entry passes (`docs/submission` = pass)
- Done: README.md, docs/brief.md (moved), Dockerfile, .dockerignore. Owner decisions are recorded and ownerPending is cleared. Commit: see `git log -1`.
- Proof: init.sh is green, all 11 README links resolve, `--help` gives exit 0, and the Docker image (8.74 MB) on the real file
  is identical to results/.
- Next: the owner pushes to a GitHub repo and emails the link (the brief's submission step). Nothing is left in the ledger.
- Watch out: the PROMPTS.md rule still applies, so append any further prompt verbatim before the final push. If code changes,
  rerun `bash scripts/benchmark.sh` so that README numbers and results/ stay true.

## 2026-11-21 (7) — benchmark + committed results (`perf/benchmark` = pass)
- Done: scripts/benchmark.sh, docs/benchmark.log, results/*.csv. Commit: see `git log -1`.
- Proof: 5 runs with a median of 4.60s and ~2.2 MB RSS (VmHWM 2304 kB). init.sh is green.
- Next: `docs/submission`. FIRST get the owner's answers on the 8 `ownerPending` items, because the README states the final rules.
  If the number format or tie-break changes, update the goldens and rerun `bash scripts/benchmark.sh` so that results/ stays current.
- Watch out: the brief must move to docs/brief.md before README.md is rewritten. Dockerfile: Go build stage + a distroless/static run stage.

## 2026-11-21 (6) — top10_cpa.csv; all output features done (`agg/top10-cpa` = pass)
- Done: TopByCPA (excludes conversions = 0, CPA asc, id asc). The CLI writes both reports from one table. Commit: see `git log -1`.
- Proof: `bash init.sh` is green and gate [5] diffs all 3 goldens. The real top10_cpa is identical to an awk ranking (4.55s, 2 MB).
- Next: `perf/benchmark`. Run `/usr/bin/time -v` on ad_data.csv, write results/ + docs/benchmark.log, and cross-check RSS with VmHWM.
  Then `docs/submission`.
- Watch out: there are 7 open `ownerPending` items. Ask the owner before docs/submission, since the README must state the final rules.
  Start-up scripts in the scratchpad are stale, so never rerun an old one blindly (one flipped top10-ctr back to in-progress this session; restored).

## 2026-11-21 (5) — top10_ctr.csv (`agg/top10-ctr` = pass)
- Done: internal/report (rank + CSV writer); the CLI writes top10_ctr.csv. Commit: see `git log -1`.
- Proof: `bash init.sh` is green and gate [5] covers top10_ctr. The real output is identical to an awk ranking (4.77s, 2.1 MB).
- Next: `agg/top10-cpa`. Add TopByCPA (exclude conversions = 0, CPA asc, id asc), `TopCPAFile`, a golden file,
  and the awk cross-check (ctr-check pattern, progress session 5).
- Watch out: at 4 dp, all 10 real top-CTR rows print 0.0275. The owner should decide the format (`ownerPending` on top10-ctr).
