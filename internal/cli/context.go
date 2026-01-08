package cli

import (
	"fmt"
	"os"

	"github.com/gwuah/piko/internal/state"
)

type Context struct {
	DB      *state.DB
	Project *state.Project
	CWD     string
}

func NewContext() (*Context, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	db, projectRoot, err := state.OpenLocal(cwd)
	if err != nil {
		return nil, err
	}

	if err := db.Initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	project, err := db.GetProjectByPath(projectRoot)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("project data corrupted: %w", err)
	}

	return &Context{
		DB:      db,
		Project: project,
		CWD:     cwd,
	}, nil
}

func NewContextWithoutProject() (*Context, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	db, _, err := state.OpenLocal(cwd)
	if err != nil {
		return nil, err
	}

	if err := db.Initialize(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &Context{
		DB:  db,
		CWD: cwd,
	}, nil
}

func (c *Context) Close() {
	if c.DB != nil {
		c.DB.Close()
	}
}

func (c *Context) GetEnvironment(name string) (*state.Environment, error) {
	return c.DB.GetEnvironmentByName(c.Project.ID, name)
}

func (c *Context) ListEnvironments() ([]*state.Environment, error) {
	return c.DB.ListEnvironmentsByProject(c.Project.ID)
}

func (c *Context) EnvironmentExists(name string) (bool, error) {
	return c.DB.EnvironmentExists(c.Project.ID, name)
}

func (c *Context) DeleteEnvironment(name string) error {
	return c.DB.DeleteEnvironment(c.Project.ID, name)
}
