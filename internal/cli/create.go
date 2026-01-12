package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gwuah/piko/internal/operations"
	"github.com/gwuah/piko/internal/state"
	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:         "create <name>",
	Short:       "Create a new worktree environment",
	Long:        "Create a new worktree environment. Use project/name syntax to create in a specific project from anywhere.",
	Args:        cobra.ExactArgs(1),
	RunE:        runCreate,
	Annotations: Requires(ToolGit, ToolTmux),
}

var (
	createBranch   string
	createNoAttach bool
)

func init() {
	envCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createBranch, "branch", "", "Base branch to create the new branch from")
	createCmd.Flags().BoolVar(&createNoAttach, "no-attach", false, "Don't attach to tmux session after creation")
}

func runCreate(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	arg := args[0]

	var db *state.DB
	var project *state.Project
	var name string
	var err error

	if strings.Contains(arg, "/") {
		parts := strings.SplitN(arg, "/", 2)
		projectName := parts[0]
		name = parts[1]

		db, err = state.OpenCentral()
		if err != nil {
			return err
		}
		defer db.Close()

		if err := db.Initialize(); err != nil {
			return err
		}

		project, err = db.GetProjectByName(projectName)
		if err != nil {
			return fmt.Errorf("project %q not found", projectName)
		}
	} else {
		name = arg
		ctx, err := NewContext()
		if err == nil {
			defer ctx.Close()
			db = ctx.DB
			project = ctx.Project
		} else {
			project, db, err = selectProject()
			if err != nil {
				return err
			}
			defer db.Close()
		}
	}

	api := NewAPIClient()
	if api.IsServerRunning() {
		if err := api.CreateEnvironment(project.ID, name, createBranch); err == nil {
			sessionName := tmux.SessionName(project.Name, name)
			if !createNoAttach && tmux.SessionExists(sessionName) {
				return tmux.Attach(sessionName)
			}
			return nil
		}
	}

	result, err := operations.CreateEnvironment(operations.CreateEnvironmentOptions{
		DB:      db,
		Project: project,
		Name:    name,
		Branch:  createBranch,
		Logger:  &operations.StdoutLogger{},
	})
	if err != nil {
		return err
	}

	if !createNoAttach && tmux.SessionExists(result.SessionName) {
		return tmux.Attach(result.SessionName)
	}

	return nil
}

func selectProject() (*state.Project, *state.DB, error) {
	db, err := state.OpenCentral()
	if err != nil {
		return nil, nil, err
	}

	if err := db.Initialize(); err != nil {
		db.Close()
		return nil, nil, err
	}

	projects, err := db.ListProjects()
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		db.Close()
		return nil, nil, fmt.Errorf("no projects registered (run 'piko init' in a project first)")
	}

	if len(projects) == 1 {
		return projects[0], db, nil
	}

	if _, err := exec.LookPath("fzf"); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("not in a piko project. Use project/name syntax or install fzf for interactive selection")
	}

	var names []string
	for _, p := range projects {
		names = append(names, p.Name)
	}

	fzf := exec.Command("fzf", "--height=~5", "--layout=reverse-list", "--no-info", "--no-separator", "--pointer=>", "--prompt=")
	fzf.Stdin = strings.NewReader(strings.Join(names, "\n"))
	fzf.Stderr = os.Stderr

	output, err := fzf.Output()
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("cancelled")
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		db.Close()
		return nil, nil, fmt.Errorf("cancelled")
	}

	project, err := db.GetProjectByName(selected)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("project %q not found", selected)
	}

	return project, db, nil
}
