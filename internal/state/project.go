package state

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
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

func (p *Project) WorktreesDir() string {
	return filepath.Join(p.RootPath, ".piko", "worktrees")
}

func (p *Project) ComposeFullDir() string {
	if p.ComposeDir == "" {
		return p.RootPath
	}
	return filepath.Join(p.RootPath, p.ComposeDir)
}

func (db *DB) InsertProject(p *Project) error {
	result, err := db.conn.Exec(
		`INSERT INTO projects (name, root_path, compose_file, compose_dir) VALUES (?, ?, ?, ?)`,
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

func (db *DB) GetProjectByPath(path string) (*Project, error) {
	cleanPath := filepath.Clean(path)

	row := db.conn.QueryRow(
		`SELECT id, name, root_path, compose_file, COALESCE(compose_dir, ''), created_at
		 FROM projects WHERE root_path = ?`,
		cleanPath,
	)

	var p Project
	err := row.Scan(&p.ID, &p.Name, &p.RootPath, &p.ComposeFile, &p.ComposeDir, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no project found at %s", path)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &p, nil
}

func (db *DB) GetProjectByID(id int64) (*Project, error) {
	row := db.conn.QueryRow(
		`SELECT id, name, root_path, compose_file, COALESCE(compose_dir, ''), created_at
		 FROM projects WHERE id = ?`,
		id,
	)

	var p Project
	err := row.Scan(&p.ID, &p.Name, &p.RootPath, &p.ComposeFile, &p.ComposeDir, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &p, nil
}

func (db *DB) FindProjectByPath(path string) (*Project, error) {
	cleanPath := filepath.Clean(path)

	for {
		project, err := db.GetProjectByPath(cleanPath)
		if err == nil {
			return project, nil
		}

		parent := filepath.Dir(cleanPath)
		if parent == cleanPath {
			break
		}
		cleanPath = parent
	}

	return nil, fmt.Errorf("not in a piko project (run 'piko init' first)")
}

func (db *DB) ListProjects() ([]*Project, error) {
	rows, err := db.conn.Query(
		`SELECT id, name, root_path, compose_file, COALESCE(compose_dir, ''), created_at
		 FROM projects ORDER BY name ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var p Project
		err := rows.Scan(&p.ID, &p.Name, &p.RootPath, &p.ComposeFile, &p.ComposeDir, &p.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, &p)
	}

	return projects, rows.Err()
}

func (db *DB) ProjectExistsByPath(path string) (bool, error) {
	cleanPath := filepath.Clean(path)
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM projects WHERE root_path = ?`, cleanPath).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check project existence: %w", err)
	}
	return count > 0, nil
}

func (db *DB) DeleteProject(rootPath string) error {
	cleanPath := filepath.Clean(rootPath)
	result, err := db.conn.Exec(`DELETE FROM projects WHERE root_path = ?`, cleanPath)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project not found")
	}

	return nil
}

func (db *DB) DeleteProjectByName(name string) error {
	result, err := db.conn.Exec(`DELETE FROM projects WHERE LOWER(name) = LOWER(?)`, name)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("project %q not found", name)
	}

	return nil
}

func (db *DB) GetProjectByName(name string) (*Project, error) {
	row := db.conn.QueryRow(
		`SELECT id, name, root_path, compose_file, COALESCE(compose_dir, ''), created_at
		 FROM projects WHERE LOWER(name) = LOWER(?)`,
		strings.ToLower(name),
	)

	var p Project
	err := row.Scan(&p.ID, &p.Name, &p.RootPath, &p.ComposeFile, &p.ComposeDir, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("project %q not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &p, nil
}
