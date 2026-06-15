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

By default the pipeline reads from `data/raw` and writes to `data/etl.db`.
Override either with flags:

```bash
go run ./cmd/etl -source path/to/csvs -db path/to/output.db
```

Expected output against the sample data:

```
done: read 27, skipped 6, loaded 21 -> data/etl.db
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
data/raw/           sample input CSV files
```
