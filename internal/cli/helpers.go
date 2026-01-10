package cli

import (
	"fmt"
	"os"
	"os/exec"
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

func GetEnvNameOrSelect(args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	return SelectEnvironment()
}

func SelectEnvironment() (string, error) {
	if _, err := exec.LookPath("fzf"); err != nil {
		return "", fmt.Errorf("fzf not found (install with: brew install fzf)")
	}

	ctx, err := NewContextWithoutProject()
	if err != nil {
		return "", err
	}
	defer ctx.Close()

	projects, err := ctx.DB.ListProjects()
	if err != nil {
		return "", fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		return "", fmt.Errorf("no projects registered (run 'piko init' in a project first)")
	}

	var envNames []string
	for _, project := range projects {
		environments, err := ctx.DB.ListEnvironmentsByProject(project.ID)
		if err != nil {
			continue
		}
		for _, env := range environments {
			envNames = append(envNames, fmt.Sprintf("%s/%s", project.Name, env.Name))
		}
	}

	if len(envNames) == 0 {
		return "", fmt.Errorf("no environments found (create one with: piko create <name>)")
	}

	if len(envNames) == 1 {
		return envNames[0], nil
	}

	fzf := exec.Command("fzf", "--height=~10", "--layout=reverse-list", "--no-info", "--no-separator", "--pointer=>", "--prompt=env> ")
	fzf.Stdin = strings.NewReader(strings.Join(envNames, "\n"))
	fzf.Stderr = os.Stderr

	output, err := fzf.Output()
	if err != nil {
		return "", fmt.Errorf("cancelled")
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return "", fmt.Errorf("cancelled")
	}

	return selected, nil
}
