package state

import (
	"database/sql"
	"fmt"
	"time"
)

type Environment struct {
	ID            int64
	ProjectID     int64
	Name          string
	Branch        string
	Path          string
	DockerProject string
	TmuxSession   sql.NullString
	CreatedAt     time.Time
}

func (db *DB) InsertEnvironment(e *Environment) (int64, error) {
	result, err := db.conn.Exec(
		`INSERT INTO environments (project_id, name, branch, path, docker_project, tmux_session)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.ProjectID, e.Name, e.Branch, e.Path, e.DockerProject, e.TmuxSession,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert environment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

func (db *DB) GetEnvironmentByName(projectID int64, name string) (*Environment, error) {
	row := db.conn.QueryRow(
		`SELECT id, project_id, name, branch, path, docker_project, tmux_session, created_at
		 FROM environments WHERE project_id = ? AND name = ?`,
		projectID, name,
	)

	var e Environment
	err := row.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Branch, &e.Path, &e.DockerProject, &e.TmuxSession, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment %q not found", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	return &e, nil
}

func (db *DB) GetEnvironmentByID(id int64) (*Environment, error) {
	row := db.conn.QueryRow(
		`SELECT id, project_id, name, branch, path, docker_project, tmux_session, created_at
		 FROM environments WHERE id = ?`,
		id,
	)

	var e Environment
	err := row.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Branch, &e.Path, &e.DockerProject, &e.TmuxSession, &e.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("environment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	return &e, nil
}

func (db *DB) ListEnvironmentsByProject(projectID int64) ([]*Environment, error) {
	rows, err := db.conn.Query(
		`SELECT id, project_id, name, branch, path, docker_project, tmux_session, created_at
		 FROM environments WHERE project_id = ? ORDER BY created_at DESC`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	defer rows.Close()

	var environments []*Environment
	for rows.Next() {
		var e Environment
		err := rows.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Branch, &e.Path, &e.DockerProject, &e.TmuxSession, &e.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}
		environments = append(environments, &e)
	}

	return environments, rows.Err()
}

func (db *DB) ListAllEnvironments() ([]*Environment, error) {
	rows, err := db.conn.Query(
		`SELECT id, project_id, name, branch, path, docker_project, tmux_session, created_at
		 FROM environments ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list environments: %w", err)
	}
	defer rows.Close()

	var environments []*Environment
	for rows.Next() {
		var e Environment
		err := rows.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Branch, &e.Path, &e.DockerProject, &e.TmuxSession, &e.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}
		environments = append(environments, &e)
	}

	return environments, rows.Err()
}

func (db *DB) EnvironmentExists(projectID int64, name string) (bool, error) {
	var count int
	err := db.conn.QueryRow(
		`SELECT COUNT(*) FROM environments WHERE project_id = ? AND name = ?`,
		projectID, name,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check environment existence: %w", err)
	}
	return count > 0, nil
}

func (db *DB) DeleteEnvironment(projectID int64, name string) error {
	result, err := db.conn.Exec(
		`DELETE FROM environments WHERE project_id = ? AND name = ?`,
		projectID, name,
	)
	if err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("environment %q not found", name)
	}

	return nil
}

type EnvironmentWithProject struct {
	Environment *Environment
	Project     *Project
}

func (db *DB) FindEnvironmentGlobally(name string) ([]EnvironmentWithProject, error) {
	rows, err := db.conn.Query(
		`SELECT e.id, e.project_id, e.name, e.branch, e.path, e.docker_project, e.tmux_session, e.created_at,
		        p.id, p.name, p.root_path, p.compose_file, COALESCE(p.compose_dir, ''), p.created_at
		 FROM environments e
		 JOIN projects p ON e.project_id = p.id
		 WHERE e.name = ?
		 ORDER BY e.created_at DESC`,
		name,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find environment: %w", err)
	}
	defer rows.Close()

	var results []EnvironmentWithProject
	for rows.Next() {
		var e Environment
		var p Project
		err := rows.Scan(
			&e.ID, &e.ProjectID, &e.Name, &e.Branch, &e.Path, &e.DockerProject, &e.TmuxSession, &e.CreatedAt,
			&p.ID, &p.Name, &p.RootPath, &p.ComposeFile, &p.ComposeDir, &p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan environment: %w", err)
		}
		results = append(results, EnvironmentWithProject{Environment: &e, Project: &p})
	}

	return results, rows.Err()
}
