# sensor-etl

A small command-line **ETL pipeline** for weather/IoT sensor readings, written in Go.

It reads raw sensor data from CSV files, validates and normalizes each reading,
and loads the clean results into a SQLite database.

```
  CSV files            normalize units            SQLite
 (data/raw)   ──▶   validate & clean   ──▶   (data/etl.db)
   extract              transform                 load
```

## What it does

- **Extract** — reads every `*.csv` file in a source directory.
- **Transform** — for each reading it:
  - parses timestamps (several common formats are accepted),
  - converts units to a canonical form (°F/K → °C, Pa/kPa → hPa),
  - rejects records that are malformed or physically implausible
    (e.g. humidity above 100%, an unparseable timestamp, an unknown metric).
- **Load** — writes the normalized readings into a `readings` table in SQLite.

Records that fail validation are skipped and logged; the run still completes.

## Requirements

- Go 1.26 or newer.

That's it — the SQLite driver is pure Go ([`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite)),
so there is no CGO toolchain or external database to install.

## Setup & running

```bash
# Fetch dependencies
go mod download

# Run the pipeline against the sample data
go run ./cmd/etl

# Or build a binary first
go build -o etl ./cmd/etl
./etl
```

By default the pipeline reads from `data/raw`, writes to `data/etl.db`, and
writes a JSON run report to `data/run-report.json`. Override any of them with flags:

```bash
go run ./cmd/etl -source path/to/csvs -db path/to/output.db -report path/to/run-report.json
```

Expected output against the sample data:

```
done: read 27, skipped 6, loaded 21 -> data/etl.db (skipped: out_of_range=2 bad_timestamp=1 bad_value=1 missing_sensor_id=1 unknown_metric=1) report: data/run-report.json
```

The summary line goes to stdout. Each skipped record is also logged to stderr as
`skipping <file>:<line>: <reason>` so you can find and fix the bad row. The
`(skipped: …)` group lists reasons by count (most first) and is left out when
nothing was skipped.

## Run report

After each successful run the pipeline writes a JSON report for downstream
systems to ingest. The default path is `data/run-report.json`; change it with
`-report`. Parent directories are created as needed.

- **Overwrite policy:** the file is replaced on every run. It is written to a
  temp file in the same directory and renamed into place, so readers never see a
  partial file. To keep history, have your scheduler pass a per-run path to
  `-report`.
- **Failed runs:** if extract or load fails, no report is written. If the report
  itself can't be written, the run exits 1. The database load has already
  committed at that point, and the error message says so.

Example:

```json
{
  "schema_version": 1,
  "started_at": "2026-10-02T14:30:00Z",
  "finished_at": "2026-10-02T14:30:00.184Z",
  "duration_ms": 184,
  "source_dir": "data/raw",
  "db_path": "data/etl.db",
  "counts": {
    "read": 27,
    "skipped": 6,
    "loaded": 21
  },
  "skip_reasons": {
    "bad_timestamp": 1,
    "bad_value": 1,
    "missing_sensor_id": 1,
    "out_of_range": 2,
    "unknown_metric": 1
  }
}
```

| Field | Type | Notes |
|---|---|---|
| `schema_version` | int | Currently `1`. Reject versions you don't know. |
| `started_at` / `finished_at` | RFC 3339 string, UTC | Fractional seconds appear only when non-zero. |
| `duration_ms` | int | `finished_at - started_at` in milliseconds. |
| `source_dir` / `db_path` | string | Exactly as passed on the command line (not made absolute). |
| `counts` | object | `read`, `skipped`, `loaded`; always present. `read = skipped + loaded`. |
| `skip_reasons` | object `reason → count` | Always present (`{}` when nothing was skipped). Only reasons with a non-zero count appear. Counts sum to `counts.skipped`. |

Skip reasons are `unknown_metric`, `missing_sensor_id`, `bad_timestamp`,
`bad_value`, `unknown_unit`, `out_of_range`, and `other` (a fallback for
uncategorized errors, which shouldn't occur in practice).

**Schema contract:** renaming or removing a field or a skip reason key is a
breaking change and bumps `schema_version`. Adding a field or a new skip reason
is not, so consumers must ignore unknown fields and reason keys.

Reading it with `jq`:

```bash
# Skip rate as a percentage
jq '.counts.skipped / .counts.read * 100' data/run-report.json
```

## Inspecting the results

If you have the `sqlite3` CLI installed:

```bash
sqlite3 data/etl.db "SELECT sensor_id, metric, value, unit FROM readings LIMIT 10;"
```

## Tests

```bash
go test ./...
```

## Project layout

```
cmd/etl/            command-line entry point
internal/model/     core data types (RawReading, Reading)
internal/extract/   reading source files
internal/transform/ validation & unit normalization
internal/load/      SQLite loader
internal/pipeline/  wires the stages together
internal/report/    JSON run report
data/raw/           sample input CSV files
```
