// Package load writes normalized readings into a SQLite database.
package load

import (
	"database/sql"
	"fmt"

	"github.com/cl-wregelmann/etl-pipeline/internal/model"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGO required
)

const schema = `
CREATE TABLE IF NOT EXISTS readings (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    sensor_id  TEXT    NOT NULL,
    timestamp  TEXT    NOT NULL,
    metric     TEXT    NOT NULL,
    value      REAL    NOT NULL,
    unit       TEXT    NOT NULL
);`

// Loader writes readings to a SQLite database file.
type Loader struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite database at path and ensures
// the schema exists. Call Close when done.
func Open(path string) (*Loader, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &Loader{db: db}, nil
}

// Close releases the database handle.
func (l *Loader) Close() error {
	return l.db.Close()
}

// Insert writes all readings in a single transaction and returns the number of
// rows inserted.
func (l *Loader) Insert(readings []model.Reading) (int, error) {
	tx, err := l.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO readings (sensor_id, timestamp, metric, value, unit) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	n := 0
	for _, r := range readings {
		if _, err := stmt.Exec(r.SensorID, r.Timestamp.Format("2006-01-02T15:04:05Z07:00"), r.Metric, r.Value, r.Unit); err != nil {
			return n, fmt.Errorf("insert reading for %q: %w", r.SensorID, err)
		}
		n++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return n, nil
}
