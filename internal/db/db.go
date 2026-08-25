// Package db provides the local SQLite library for stored chord charts and
// setlists.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// ErrExists is returned when creating a setlist with a name that is already
// taken.
var ErrExists = errors.New("setlist name already exists")

// ErrNotFound is returned when operating on a chart (or setlist) id that does
// not exist.
var ErrNotFound = errors.New("chart not found")

// Store is a handle to the chart library database.
type Store struct {
	db *sql.DB
}

type schemaMigration struct {
	version int
	stmts   []string
}

var migrations = []schemaMigration{
	{
		version: 1,
		stmts: []string{
			`CREATE TABLE IF NOT EXISTS charts (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				title TEXT NOT NULL DEFAULT '',
				artist TEXT NOT NULL DEFAULT '',
				source TEXT NOT NULL DEFAULT '',
				content BLOB NOT NULL,
				created_at DATETIME NOT NULL,
				updated_at DATETIME NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS setlists (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				created_at DATETIME NOT NULL
			)`,
			`CREATE TABLE IF NOT EXISTS setlist_items (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				setlist_id INTEGER NOT NULL REFERENCES setlists(id) ON DELETE CASCADE,
				position INTEGER NOT NULL,
				chart_id INTEGER NOT NULL REFERENCES charts(id) ON DELETE CASCADE,
				UNIQUE(setlist_id, position)
			)`,
		},
	},
}

// DefaultPath resolves the library database location:
// $XDG_DATA_HOME/chart-tty/library.db, defaulting to
// ~/.local/share/chart-tty/library.db.
func DefaultPath() (string, error) {
	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "chart-tty", "library.db"), nil
}

// Open opens (creating if necessary) the library at path and applies pending
// migrations.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating library directory: %w", err)
	}
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening library %s: %w", path, err)
	}
	// modernc/sqlite handles concurrent access within one connection best.
	sqlDB.SetMaxOpenConns(1)
	// Enforce foreign keys (ON DELETE CASCADE) on this connection.
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}
	s := &Store{db: sqlDB}
	if err := s.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return s, nil
}

// OpenMemory opens a transient in-memory library, mainly for tests.
func OpenMemory() (*Store, error) {
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		sqlDB.Close()
		return nil, err
	}
	s := &Store{db: sqlDB}
	if err := s.migrate(); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	if _, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("creating schema_version: %w", err)
	}
	var current int
	row := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}
	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		for _, stmt := range m.stmts {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("migration %d: %w", m.version, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO schema_version (version) VALUES (?)`, m.version); err != nil {
			tx.Rollback()
			return fmt.Errorf("recording migration %d: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }

// ChartMeta describes a stored chart without its content.
type ChartMeta struct {
	ID        int64
	Title     string
	Artist    string
	Source    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StoredChart is a full chart row including its raw content.
type StoredChart struct {
	ChartMeta
	Content string
}

// AddChart stores a chart and returns its id. Content is stored verbatim as a
// blob; title/artist are metadata supplied by the caller.
func (s *Store) AddChart(title, artist, source, content string) (int64, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(
		`INSERT INTO charts (title, artist, source, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		title, artist, source, content, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("inserting chart: %w", err)
	}
	return res.LastInsertId()
}

// GetChart fetches one chart by id.
func (s *Store) GetChart(id int64) (StoredChart, error) {
	var c StoredChart
	var created, updated time.Time
	err := s.db.QueryRow(
		`SELECT id, title, artist, source, content, created_at, updated_at FROM charts WHERE id = ?`, id,
	).Scan(&c.ID, &c.Title, &c.Artist, &c.Source, &c.Content, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, fmt.Errorf("querying chart %d: %w", id, err)
	}
	c.CreatedAt, c.UpdatedAt = created.UTC(), updated.UTC()
	return c, nil
}

// DeleteChart removes a chart and its setlist memberships. The id is removed
// from every setlist explicitly (belt-and-suspenders on top of the FK
// cascade).
func (s *Store) DeleteChart(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("starting delete: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM setlist_items WHERE chart_id = ?`, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("removing setlist memberships: %w", err)
	}
	res, err := tx.Exec(`DELETE FROM charts WHERE id = ?`, id)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("deleting chart %d: %w", id, err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		tx.Rollback()
		return ErrNotFound
	}
	return tx.Commit()
}

// ListCharts returns all stored charts, newest first.
func (s *Store) ListCharts() ([]ChartMeta, error) {
	rows, err := s.db.Query(
		`SELECT id, title, artist, source, created_at, updated_at FROM charts ORDER BY updated_at DESC, id DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing charts: %w", err)
	}
	defer rows.Close()

	var out []ChartMeta
	for rows.Next() {
		var c ChartMeta
		var created, updated time.Time
		if err := rows.Scan(&c.ID, &c.Title, &c.Artist, &c.Source, &created, &updated); err != nil {
			return nil, fmt.Errorf("scanning chart row: %w", err)
		}
		c.CreatedAt, c.UpdatedAt = created.UTC(), updated.UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

// SetlistMeta describes a stored setlist.
type SetlistMeta struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

// CreateSetlist inserts a new empty setlist. Names are unique.
func (s *Store) CreateSetlist(name string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO setlists (name, created_at) VALUES (?, ?)`,
		name, time.Now().UTC(),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrExists
		}
		return 0, fmt.Errorf("inserting setlist: %w", err)
	}
	return res.LastInsertId()
}

// ListSetlists returns all setlists in creation order.
func (s *Store) ListSetlists() ([]SetlistMeta, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM setlists ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("listing setlists: %w", err)
	}
	defer rows.Close()

	var out []SetlistMeta
	for rows.Next() {
		var sl SetlistMeta
		var created time.Time
		if err := rows.Scan(&sl.ID, &sl.Name, &created); err != nil {
			return nil, fmt.Errorf("scanning setlist row: %w", err)
		}
		sl.CreatedAt = created.UTC()
		out = append(out, sl)
	}
	return out, rows.Err()
}

// AppendSetlistItem adds a chart to the end of a setlist.
func (s *Store) AppendSetlistItem(setlistID, chartID int64) error {
	var next int
	err := s.db.QueryRow(
		`SELECT COALESCE(MAX(position), 0) + 1 FROM setlist_items WHERE setlist_id = ?`,
		setlistID,
	).Scan(&next)
	if err != nil {
		return fmt.Errorf("computing next position: %w", err)
	}
	if _, err := s.db.Exec(
		`INSERT INTO setlist_items (setlist_id, position, chart_id) VALUES (?, ?, ?)`,
		setlistID, next, chartID,
	); err != nil {
		return fmt.Errorf("appending setlist item: %w", err)
	}
	return nil
}

// SetlistCharts returns the charts of a setlist in performance order.
func (s *Store) SetlistCharts(setlistID int64) ([]StoredChart, error) {
	rows, err := s.db.Query(
		`SELECT c.id, c.title, c.artist, c.source, c.content, c.created_at, c.updated_at
		 FROM setlist_items si
		 JOIN charts c ON c.id = si.chart_id
		 WHERE si.setlist_id = ?
		 ORDER BY si.position`,
		setlistID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying setlist charts: %w", err)
	}
	defer rows.Close()

	var out []StoredChart
	for rows.Next() {
		var c StoredChart
		var created, updated time.Time
		if err := rows.Scan(&c.ID, &c.Title, &c.Artist, &c.Source, &c.Content, &created, &updated); err != nil {
			return nil, fmt.Errorf("scanning setlist chart: %w", err)
		}
		c.CreatedAt, c.UpdatedAt = created.UTC(), updated.UTC()
		out = append(out, c)
	}
	return out, rows.Err()
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure
// from the pure-Go driver.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	type coder interface{ Code() int }
	var c coder
	if errors.As(err, &c) {
		const sqliteConstraintUnique = 2067 // SQLITE_CONSTRAINT_UNIQUE
		if c.Code() == sqliteConstraintUnique {
			return true
		}
	}
	return false
}
