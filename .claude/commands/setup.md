---
description: Set up the sensor-etl pipeline locally and run it against the sample data
---

Set up this Go ETL pipeline for local development. Run each step in order and stop if any step fails.

1. Confirm Go is installed and recent enough: run `go version` and check it reports Go 1.26 or newer. If not, tell the user to install/upgrade Go and stop.
2. Run `go mod download` to fetch dependencies. The SQLite driver is pure Go (`modernc.org/sqlite`), so no C compiler or `CGO_ENABLED` is required.
3. Build everything to confirm the project compiles: `go build ./...`.
4. Run the test suite: `go test ./...`. All tests should pass.
5. Run the pipeline against the bundled sample data: `go run ./cmd/etl`. It reads from `data/raw` and writes to `data/etl.db`. Expect a summary line like `done: read 27, skipped 6, loaded 21 -> data/etl.db (skipped: …) report: data/run-report.json`, plus a JSON run report at `data/run-report.json` — some sample rows are intentionally malformed and are skipped, which is expected.
6. Tell the user setup is complete. Mention they can:
   - re-run the pipeline with `go run ./cmd/etl` (or build a binary with `go build -o etl ./cmd/etl`),
   - point it at other data with `-source`, `-db`, and `-report` flags,
   - inspect the loaded rows with `sqlite3 data/etl.db "SELECT * FROM readings LIMIT 10;"` if the `sqlite3` CLI is installed.
