package state

import (
	"database/sql"
	"fmt"
	"time"
)

type Project struct {
	ID          int64
	Name        string
	RootPath    string
	ComposeFile string
	CreatedAt   time.Time
}

func (db *DB) InsertProject(p *Project) error {
	result, err := db.conn.Exec(
		`INSERT INTO project (name, root_path, compose_file) VALUES (?, ?, ?)`,
		p.Name, p.RootPath, p.ComposeFile,
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
		`SELECT id, name, root_path, compose_file, created_at FROM project LIMIT 1`,
	)

	var p Project
	err := row.Scan(&p.ID, &p.Name, &p.RootPath, &p.ComposeFile, &p.CreatedAt)
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
