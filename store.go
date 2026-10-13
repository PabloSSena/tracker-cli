package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Note is a single day's note.
type Note struct {
	Day  string // YYYY-MM-DD format
	Body string
}

// Preview returns the first non-empty line of the body, for the list view.
func (n Note) Preview() string { return firstLine(n.Body) }

// Pinned is a pinned note: it lives until deleted, independent of days.
type Pinned struct {
	ID    int64
	Title string
	Body  string
}

func (p Pinned) Preview() string { return firstLine(p.Body) }

// firstLine returns the first non-empty line of s.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

type Store struct {
	conn *sql.DB
}

// dbPath resolves the database path at ~/.tracker/notes.db, creating the folder.
func dbPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".tracker")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "notes.db"), nil
}

func (s *Store) Init() error {
	path, err := dbPath()
	if err != nil {
		return err
	}

	s.conn, err = sql.Open("sqlite", path)
	if err != nil {
		return err
	}

	createTableQuery := `CREATE TABLE IF NOT EXISTS notes (
		day        TEXT PRIMARY KEY,
		body       TEXT NOT NULL DEFAULT '',
		updated_at TEXT NOT NULL
	);`

	if _, err := s.conn.Exec(createTableQuery); err != nil {
		return err
	}

	createPinnedQuery := `CREATE TABLE IF NOT EXISTS pinned (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		title      TEXT NOT NULL,
		body       TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`

	if _, err := s.conn.Exec(createPinnedQuery); err != nil {
		return err
	}

	return nil
}

// ListDays returns all notes, newest day first.
func (s *Store) ListDays() ([]Note, error) {
	rows, err := s.conn.Query("SELECT day, body FROM notes ORDER BY day DESC;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.Day, &note.Body); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

// GetDay returns a day's note body (empty string if it doesn't exist).
func (s *Store) GetDay(day string) (string, error) {
	var body string
	err := s.conn.QueryRow("SELECT body FROM notes WHERE day = ?;", day).Scan(&body)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return body, nil
}

// UpsertDay inserts or updates a day's note.
func (s *Store) UpsertDay(day, body string) error {
	upsertQuery := `INSERT INTO notes (day, body, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT (day) DO UPDATE SET body=excluded.body, updated_at=excluded.updated_at;`

	now := nowUTC()
	if _, err := s.conn.Exec(upsertQuery, day, body, now); err != nil {
		return err
	}
	return nil
}

// nowUTC uses a fixed width (zero-padded nanos) so that sorting updated_at as
// text is chronological — RFC3339Nano trims zeros and would break that.
func nowUTC() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z") }

// ListPinned returns pinned notes, most recently edited first.
func (s *Store) ListPinned() ([]Pinned, error) {
	rows, err := s.conn.Query("SELECT id, title, body FROM pinned ORDER BY updated_at DESC, id DESC;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pinned := []Pinned{}
	for rows.Next() {
		var p Pinned
		if err := rows.Scan(&p.ID, &p.Title, &p.Body); err != nil {
			return nil, err
		}
		pinned = append(pinned, p)
	}
	return pinned, rows.Err()
}

// CreatePinned creates a pinned note and returns its id.
func (s *Store) CreatePinned(title, body string) (int64, error) {
	now := nowUTC()
	res, err := s.conn.Exec(
		"INSERT INTO pinned (title, body, created_at, updated_at) VALUES (?, ?, ?, ?);",
		title, body, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdatePinnedBody updates a pinned note's body.
func (s *Store) UpdatePinnedBody(id int64, body string) error {
	_, err := s.conn.Exec("UPDATE pinned SET body = ?, updated_at = ? WHERE id = ?;", body, nowUTC(), id)
	return err
}

// RenamePinned changes a pinned note's title.
func (s *Store) RenamePinned(id int64, title string) error {
	_, err := s.conn.Exec("UPDATE pinned SET title = ?, updated_at = ? WHERE id = ?;", title, nowUTC(), id)
	return err
}

// DeletePinned deletes a pinned note.
func (s *Store) DeletePinned(id int64) error {
	_, err := s.conn.Exec("DELETE FROM pinned WHERE id = ?;", id)
	return err
}
