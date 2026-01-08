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

func FindPikoRoot(startPath string) (string, error) {
	cleanPath := filepath.Clean(startPath)

	for {
		pikoDir := filepath.Join(cleanPath, ".piko")
		if info, err := os.Stat(pikoDir); err == nil && info.IsDir() {
			return cleanPath, nil
		}

		parent := filepath.Dir(cleanPath)
		if parent == cleanPath {
			break
		}
		cleanPath = parent
	}

	return "", fmt.Errorf("not in a piko project (run 'piko init' first)")
}

func LocalDBPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".piko", "state.db")
}

func OpenLocal(startPath string) (*DB, string, error) {
	projectRoot, err := FindPikoRoot(startPath)
	if err != nil {
		return nil, "", err
	}

	dbPath := LocalDBPath(projectRoot)
	db, err := Open(dbPath)
	if err != nil {
		return nil, "", err
	}

	return db, projectRoot, nil
}

func CreateLocal(projectRoot string) (*DB, error) {
	pikoDir := filepath.Join(projectRoot, ".piko")
	if err := os.MkdirAll(pikoDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .piko directory: %w", err)
	}

	dbPath := LocalDBPath(projectRoot)
	return Open(dbPath)
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if _, err := conn.Exec("PRAGMA journal_mode = WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
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
