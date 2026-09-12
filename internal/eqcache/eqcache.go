package eqcache

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type CachedEQ struct {
	DurationMs int
	HopMs      int
	NumBars    int
	Bars       []byte // flat: frame * NumBars + bar
}

func (c *CachedEQ) BarsAt(elapsedMs int) []int {
	if c == nil || c.HopMs <= 0 || c.NumBars <= 0 || len(c.Bars) == 0 {
		return nil
	}
	if elapsedMs < 0 {
		elapsedMs = 0
	}
	frame := elapsedMs / c.HopMs
	total := len(c.Bars) / c.NumBars
	if frame >= total {
		frame = total - 1
	}
	if frame < 0 {
		frame = 0
	}
	out := make([]int, c.NumBars)
	base := frame * c.NumBars
	for i := 0; i < c.NumBars; i++ {
		out[i] = int(c.Bars[base+i])
	}
	return out
}

type DB struct {
	db *sql.DB
}

func Open() (*DB, error) {
	dir := cacheDir()
	_ = os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, "eq.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := initSchema(db); err != nil {
		return nil, err
	}
	return &DB{db: db}, nil
}

func cacheDir() string {
	if c, err := os.UserCacheDir(); err == nil {
		return filepath.Join(c, "iradio")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "iradio")
}

func initSchema(db *sql.DB) error {
	_, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS track_eq (
            path        TEXT PRIMARY KEY,
            mtime_unix  INTEGER NOT NULL,
            size        INTEGER NOT NULL,
            duration_ms INTEGER NOT NULL,
            hop_ms      INTEGER NOT NULL,
            num_bars    INTEGER NOT NULL,
            bars        BLOB NOT NULL
        );
        CREATE INDEX IF NOT EXISTS idx_track_eq_mtime ON track_eq(mtime_unix);
    `)
	return err
}

// Lookup returns the cached EQ if it's still fresh (file unchanged).
func (d *DB) Lookup(path string) *CachedEQ {
	if d == nil || d.db == nil {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}

	row := d.db.QueryRow(
		`SELECT mtime_unix, size, duration_ms, hop_ms, num_bars, bars
         FROM track_eq WHERE path = ?`, path)
	var mtime, size int64
	var c CachedEQ
	if err := row.Scan(&mtime, &size, &c.DurationMs, &c.HopMs, &c.NumBars, &c.Bars); err != nil {
		return nil
	}
	// Cache invalidation: if file mtime or size changed, treat as stale.
	if mtime != info.ModTime().Unix() || size != info.Size() {
		return nil
	}
	return &c
}

func (d *DB) Store(path string, eq *CachedEQ) error {
	if d == nil || d.db == nil {
		return fmt.Errorf("eqcache: nil db")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`
        INSERT INTO track_eq(path, mtime_unix, size, duration_ms, hop_ms, num_bars, bars)
        VALUES(?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(path) DO UPDATE SET
            mtime_unix = excluded.mtime_unix,
            size       = excluded.size,
            duration_ms= excluded.duration_ms,
            hop_ms     = excluded.hop_ms,
            num_bars   = excluded.num_bars,
            bars       = excluded.bars
    `, path, info.ModTime().Unix(), info.Size(),
		eq.DurationMs, eq.HopMs, eq.NumBars, eq.Bars)
	return err
}

func (d *DB) Close() error {
	if d == nil || d.db == nil {
		return nil
	}
	return d.db.Close()
}

func (d *DB) Count() int {
	if d == nil || d.db == nil {
		return 0
	}
	var n int
	_ = d.db.QueryRow(`SELECT COUNT(*) FROM track_eq`).Scan(&n)
	return n
}
