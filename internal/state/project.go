package state

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"
)

type Project struct {
	ID          int64
	Name        string
	RootPath    string
	ComposeFile string
	ComposeDir  string
	CreatedAt   time.Time
}

func (p *Project) ComposeFullDir() string {
	if p.ComposeDir == "" {
		return p.RootPath
	}
	return filepath.Join(p.RootPath, p.ComposeDir)
}

func (db *DB) InsertProject(p *Project) error {
	result, err := db.conn.Exec(
		`INSERT INTO project (name, root_path, compose_file, compose_dir) VALUES (?, ?, ?, ?)`,
		p.Name, p.RootPath, p.ComposeFile, p.ComposeDir,
	)
	if err != nil {
		return fmt.Errorf("failed to insert project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	p.ID = id

	return nil
}

func (db *DB) GetProject() (*Project, error) {
	row := db.conn.QueryRow(
		`SELECT id, name, root_path, compose_file, COALESCE(compose_dir, ''), created_at FROM project LIMIT 1`,
	)

	var p Project
	err := row.Scan(&p.ID, &p.Name, &p.RootPath, &p.ComposeFile, &p.ComposeDir, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no project found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &p, nil
}

func (db *DB) ProjectExists() (bool, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM project`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check project existence: %w", err)
	}
	return count > 0, nil
}
