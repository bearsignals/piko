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

func (db *DB) GetEnvironmentByName(name string) (*Environment, error) {
	row := db.conn.QueryRow(
		`SELECT id, project_id, name, branch, path, docker_project, tmux_session, created_at
		 FROM environments WHERE name = ?`,
		name,
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

func (db *DB) ListEnvironments() ([]*Environment, error) {
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

func (db *DB) EnvironmentExists(name string) (bool, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM environments WHERE name = ?`, name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check environment existence: %w", err)
	}
	return count > 0, nil
}

func (db *DB) DeleteEnvironment(name string) error {
	_, err := db.conn.Exec(`DELETE FROM environments WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}
	return nil
}
