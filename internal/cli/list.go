package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/gwuah/piko/internal/docker"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all environments",
	RunE:    runList,
}

var listAll bool

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVarP(&listAll, "all", "a", false, "List environments from all projects")
}

func runList(cmd *cobra.Command, args []string) error {
	if listAll {
		return runListAll()
	}

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	environments, err := ctx.ListEnvironments()
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	if len(environments) == 0 {
		fmt.Println("No environments yet. Create one with: piko create <name>")
		return nil
	}

	table := NewTable("NAME", "STATUS", "BRANCH", "CREATED")
	for _, e := range environments {
		var status string
		if e.DockerProject == "" {
			status = "simple"
		} else {
			composeDir := e.Path
			if ctx.Project.ComposeDir != "" {
				composeDir = filepath.Join(e.Path, ctx.Project.ComposeDir)
			}
			status = string(docker.GetProjectStatus(composeDir, e.DockerProject))
		}
		table.Row(e.Name, status, e.Branch, formatAge(e.CreatedAt))
	}
	table.Flush()
	return nil
}

func runListAll() error {
	ctx, err := NewContextWithoutProject()
	if err != nil {
		return err
	}
	defer ctx.Close()

	projects, err := ctx.DB.ListProjects()
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects yet. Initialize one with: piko init")
		return nil
	}

	table := NewTable("PROJECT", "ENVIRONMENT", "STATUS", "BRANCH")
	for _, p := range projects {
		environments, err := ctx.DB.ListEnvironmentsByProject(p.ID)
		if err != nil {
			continue
		}

		if len(environments) == 0 {
			table.Row(p.Name, "(no environments)", "", "")
			continue
		}

		for _, e := range environments {
			var status string
			if e.DockerProject == "" {
				status = "simple"
			} else {
				composeDir := e.Path
				if p.ComposeDir != "" {
					composeDir = filepath.Join(e.Path, p.ComposeDir)
				}
				status = string(docker.GetProjectStatus(composeDir, e.DockerProject))
			}
			table.Row(p.Name, e.Name, status, e.Branch)
		}
	}
	table.Flush()
	return nil
}

func formatAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	}
	return fmt.Sprintf("%d days ago", int(d.Hours()/24))
}
