# Ad Performance Aggregator

A Go command-line tool for the FV-SEC001 challenge ([brief](docs/brief.md)). It streams the ~1 GB
advertising CSV once, sums the metrics per `campaign_id`, and writes two reports:

- `top10_ctr.csv`: the 10 campaigns with the highest CTR.
- `top10_cpa.csv`: the 10 campaigns with the lowest CPA. Campaigns with zero conversions are excluded.

The outputs from the real dataset are committed in [`results/`](results/).

## Results at a glance

| | |
|---|---|
| Input | `ad_data.csv`: 1,043,304,870 bytes, 26,843,544 rows, 50 campaigns |
| Processing time | **1.04 s** with 4 workers (median of 5 runs); 3.96 s single-threaded (`--workers 1`) |
| Peak memory (RSS) | **~11 MB** (11008–11264 kB), fixed by the buffer pool and independent of file size |
| Runtime dependencies | none (Go standard library only) |

Measured on an AMD Ryzen 7 7700 under WSL2 (4 vCPU, 3.8 GiB RAM) with Go 1.25.1. The full log is in
[`docs/benchmark.log`](docs/benchmark.log).

## Setup

You need **Go 1.25+** (or Docker, see below).

```bash
unzip ad_data.csv.zip            # produces ad_data.csv (~1 GB); not committed
go build -o bin/aggregator ./cmd/aggregator
```

## How to run

```bash
./bin/aggregator --input ad_data.csv --output results/
```

| Flag | Meaning |
|---|---|
| `--input` | Path to the input CSV (required) |
| `--output` | Directory for the result files. It is created if missing (required). |
| `--workers` | Number of parsing goroutines (default: the number of CPUs) |

The program writes these files to the output directory:

| File | Content |
|---|---|
| `top10_ctr.csv` | Top 10 campaigns by CTR, in the brief's column order |
| `top10_cpa.csv` | Top 10 campaigns by lowest CPA, zero-conversion campaigns excluded |
| `malformed_rows.csv` | Every rejected row: `line_number,reason,raw_line` (only the header when every row is valid) |

A one-line summary goes to stderr:

```
aggregator: 26843544 rows processed, 0 malformed rows skipped (see results/malformed_rows.csv), 50 campaigns
```

Exit codes:

| Code | Meaning |
|---|---|
| `0` | Success, or `--help` |
| `1` | I/O or data fault: missing or unreadable input, empty file, wrong header, unwritable output |
| `2` | Usage error: a missing or unknown flag, `--workers` below 1, or an unexpected argument |

### With Docker

```bash
docker build -t ad-aggregator .
docker run --rm -v "$PWD:/data" ad-aggregator --input /data/ad_data.csv --output /data/results
```

## Tests

```bash
go test -race ./...
```

- **Unit tests:** parsing and every rejection reason, aggregation and merging, ranking and tie-breaks, CSV
  formatting, and the CLI exit codes. Allocation tests assert that parsing and aggregating a row allocate
  nothing.
- **Parallel equivalence:** a generated file mixes valid rows, every kind of malformed row, LF and CRLF
  line endings, and lines longer than a chunk. It is parsed with 1, 2 and 8 workers and with tiny chunk
  sizes, so lines fall on every chunk boundary. Each run must produce exactly the single-worker stats,
  malformed rows (line numbers included) and totals. All tests run under `-race`.
- **End-to-end:** [`testdata/sample.csv`](testdata/sample.csv) has valid rows and one row for each kind of
  malformed row. The binary's output on that file must match [`testdata/golden/`](testdata/golden/) byte
  for byte; `init.sh` checks this.
- **Real-data cross-check (during development):** for all 50 campaigns, the totals and both top-10 files
  matched an independent `awk` computation exactly.

`bash init.sh` runs the full gate chain: gofmt, `go vet`, `go test -race`, build, and the golden-file diff
with both `--workers 1` and all CPUs.
To reproduce the benchmark, run `RUNS=5 bash scripts/benchmark.sh`.

## Libraries used

None beyond the Go standard library: `bufio`, `bytes`, `encoding/csv` (for writing only), `flag`,
`slices`, `strconv`, `sync`.

## Design decisions

- **One streaming pass, parallel parse, bounded memory.**
  - A reader goroutine cuts the file into 1 MiB chunks that end on a line boundary. It carries a partial
    last line over into the next chunk.
  - N workers (one per CPU by default) parse the chunks. Each worker folds rows into its own per-campaign
    map, so no locks are needed, and the maps are merged at the end.
  - Chunk buffers come from a fixed pool of 2×N buffers. Memory therefore depends on the worker count and
    the number of campaigns, never on the file size: about 11 MB for 26.8 M rows.
- **Malformed rows stay exact under parallelism.** Each worker reports its malformed rows relative to its
  chunk. A collector puts the chunks back in input order and turns those into absolute line numbers, so
  `malformed_rows.csv` matches a single-threaded run byte for byte.
- **No per-row allocation.** Lines are split on commas in place, and numbers are parsed straight from
  bytes. The campaign ID is copied only the first time it is seen (`m[string(b)]` lookups don't allocate).
  Tests enforce this with `testing.AllocsPerRun`.
- **A hand-written splitter instead of `encoding/csv` for reading.** The schema has no free-text columns
  and the data has no quotes. The splitter also keeps the original line for `malformed_rows.csv`. A quoted
  row would be rejected as malformed, never misread.
- **Money in integer cents.** Spend is parsed into `int64` cents and summed exactly, which avoids float
  drift over 26.8 M additions. It is converted to float only to compute CPA.
- **Malformed rows are skipped, counted and logged.** A row is rejected for:
  - the wrong number of fields or an empty line;
  - an empty `campaign_id`;
  - a date that is not a real `YYYY-MM-DD` date;
  - a count that is not a non-negative integer;
  - a spend that is not a non-negative amount with at most 2 decimals.

  Line numbers count the header as line 1.
- **The whole file is invalid when** it is empty or its header is not exactly
  `campaign_id,date,impressions,clicks,spend,conversions`. The program then exits with code 1. A UTF-8 BOM
  and CRLF line endings are accepted; the real file uses CRLF.
- **Rules the brief does not state:**
  - CTR is 0 when a campaign has no impressions.
  - Ties on CTR or CPA are broken by `campaign_id` ascending, so the output is deterministic.
  - CTR is printed with 4 decimals, and CPA and spend with 2, as in the brief's samples. Ranking uses full
    precision. In the real data all ten top-CTR campaigns print as `0.0275`, but their full-precision
    order is preserved.
  - CPA is an empty cell when there are no conversions. This can only happen in `top10_ctr.csv`.
- **Why parallel.** Reading the file alone takes 0.16 s with a warm page cache and 1.2 s with a cold one,
  so the run is CPU-bound in parsing. On 4 cores the parse is 3.8× faster: 1.04 s instead of 3.96 s. The cost is a fixed 8 MiB
  buffer pool. `--workers 1` gives the single-threaded behaviour.

## Project layout

```
cmd/aggregator/      main: delegates to internal/cli
internal/cli/        flags, path checks, exit codes, wiring
internal/ingest/     row validation, parallel chunk pipeline, malformed-row log
internal/agg/        per-campaign totals, CTR, CPA
internal/report/     top-10 ranking + CSV writer
testdata/            sample input + golden outputs
scripts/benchmark.sh reproducible benchmark → docs/benchmark.log
results/             outputs from the real 1 GB file
```

## AI-assisted workflow

This project was built with Claude Code. The raw prompts are in [`PROMPTS.md`](PROMPTS.md). The work was
driven by a small session harness:

- [`AGENTS.md`](AGENTS.md): the rules the agent follows.
- [`feature_list.json`](feature_list.json): each feature with the evidence that proves it.
- [`init.sh`](init.sh): the verification gates.
- [`progress.md`](progress.md) and [`session-handoff.md`](session-handoff.md): the session log.

Each feature was marked done only after the gates passed and the built binary was run against real input.
