package state

import (
	"database/sql"
	"errors"
	"fmt"
)

const projectColumns = "id, name, root_path, compose_file, COALESCE(compose_dir, ''), created_at"
const environmentColumns = "id, project_id, name, branch, path, docker_project, tmux_session, created_at"

type Scanner interface {
	Scan(dest ...any) error
}

func scanProject(s Scanner) (*Project, error) {
	var p Project
	err := s.Scan(&p.ID, &p.Name, &p.RootPath, &p.ComposeFile, &p.ComposeDir, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func scanEnvironment(s Scanner) (*Environment, error) {
	var e Environment
	err := s.Scan(&e.ID, &e.ProjectID, &e.Name, &e.Branch, &e.Path, &e.DockerProject, &e.TmuxSession, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func getOneProject(row *sql.Row, notFoundMsg string) (*Project, error) {
	p, err := scanProject(row)
	if err == sql.ErrNoRows {
		return nil, errors.New(notFoundMsg)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return p, nil
}

func getOneEnvironment(row *sql.Row, notFoundMsg string) (*Environment, error) {
	e, err := scanEnvironment(row)
	if err == sql.ErrNoRows {
		return nil, errors.New(notFoundMsg)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}
	return e, nil
}

func checkRowsAffected(result sql.Result, notFoundMsg string) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return errors.New(notFoundMsg)
	}
	return nil
}
