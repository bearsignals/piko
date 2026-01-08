package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT UNIQUE NOT NULL,
    compose_file TEXT DEFAULT 'docker-compose.yml',
    compose_dir TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS environments (
    id INTEGER PRIMARY KEY,
    project_id INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    branch TEXT NOT NULL,
    path TEXT NOT NULL,
    docker_project TEXT NOT NULL,
    tmux_session TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, name)
);
`

type DB struct {
	conn *sql.DB
	path string
}

func CentralDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	pikoDir := filepath.Join(home, ".piko")
	if err := os.MkdirAll(pikoDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create ~/.piko directory: %w", err)
	}

	return filepath.Join(pikoDir, "state.db"), nil
}

func OpenCentral() (*DB, error) {
	dbPath, err := CentralDBPath()
	if err != nil {
		return nil, err
	}
	return Open(dbPath)
}

func Open(path string) (*DB, error) {
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &DB{conn: conn, path: path}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Initialize() error {
	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}
