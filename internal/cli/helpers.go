package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gwuah/piko/internal/state"
)

type ResolvedEnvironment struct {
	Ctx         *Context
	Project     *state.Project
	Environment *state.Environment
	ComposeDir  string
}

func (r *ResolvedEnvironment) Close() {
	if r.Ctx != nil {
		r.Ctx.Close()
	}
}

func ResolveEnvironment(name string) (*ResolvedEnvironment, error) {
	ctx, err := NewContext()
	if err != nil {
		return nil, err
	}

	environment, err := ctx.GetEnvironment(name)
	if err != nil {
		ctx.Close()
		return nil, fmt.Errorf("environment %q not found", name)
	}

	composeDir := environment.Path
	if ctx.Project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, ctx.Project.ComposeDir)
	}

	return &ResolvedEnvironment{
		Ctx:         ctx,
		Project:     ctx.Project,
		Environment: environment,
		ComposeDir:  composeDir,
	}, nil
}

func ResolveEnvironmentGlobally(name string) (*ResolvedEnvironment, error) {
	ctx, err := NewContextWithoutProject()
	if err != nil {
		return nil, err
	}

	var project *state.Project
	var environment *state.Environment
	var envName string

	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		projectName, envName := parts[0], parts[1]
		project, err = ctx.DB.GetProjectByName(projectName)
		if err != nil {
			ctx.Close()
			return nil, fmt.Errorf("project %q not found", projectName)
		}
		environment, err = ctx.DB.GetEnvironmentByName(project.ID, envName)
		if err != nil {
			ctx.Close()
			return nil, fmt.Errorf("environment %q not found in project %q", envName, projectName)
		}
	} else {
		envName = name
		results, err := ctx.DB.FindEnvironmentGlobally(envName)
		if err != nil {
			ctx.Close()
			return nil, err
		}
		if len(results) == 0 {
			ctx.Close()
			return nil, fmt.Errorf("environment %q not found", envName)
		}
		if len(results) > 1 {
			ctx.Close()
			var names []string
			for _, r := range results {
				names = append(names, fmt.Sprintf("%s/%s", r.Project.Name, r.Environment.Name))
			}
			return nil, fmt.Errorf("multiple environments named %q found: %s (use project/env syntax)", envName, strings.Join(names, ", "))
		}
		project = results[0].Project
		environment = results[0].Environment
	}

	composeDir := environment.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, project.ComposeDir)
	}

	return &ResolvedEnvironment{
		Ctx:         ctx,
		Project:     project,
		Environment: environment,
		ComposeDir:  composeDir,
	}, nil
}

func RequireDocker(name string) (*ResolvedEnvironment, error) {
	resolved, err := ResolveEnvironment(name)
	if err != nil {
		return nil, err
	}

	if resolved.Environment.DockerProject == "" {
		resolved.Close()
		return nil, fmt.Errorf("environment %q is in simple mode (no Docker containers)", name)
	}

	return resolved, nil
}

func RequireDockerGlobally(name string) (*ResolvedEnvironment, error) {
	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return nil, err
	}

	if resolved.Environment.DockerProject == "" {
		resolved.Close()
		return nil, fmt.Errorf("environment %q is in simple mode (no Docker containers)", name)
	}

	return resolved, nil
}
