package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Note representa a anotação de um único dia.
type Note struct {
	Day  string // formato YYYY-MM-DD
	Body string
}

// Preview retorna a primeira linha não vazia do corpo, para exibição na lista.
func (n Note) Preview() string {
	for _, line := range strings.Split(n.Body, "\n") {
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

// dbPath resolve o caminho do banco em ~/.tracker/notes.db, criando a pasta.
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

	return nil
}

// ListDays retorna todas as anotações, do dia mais recente para o mais antigo.
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

// GetDay retorna o corpo da anotação de um dia (string vazia se não existir).
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

// UpsertDay insere ou atualiza a anotação de um dia.
func (s *Store) UpsertDay(day, body string) error {
	upsertQuery := `INSERT INTO notes (day, body, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT (day) DO UPDATE SET body=excluded.body, updated_at=excluded.updated_at;`

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.conn.Exec(upsertQuery, day, body, now); err != nil {
		return err
	}
	return nil
}
