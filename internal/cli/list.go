package cli

import (
	"fmt"
	"path/filepath"
	"text/tabwriter"
	"time"

	"os"

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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tBRANCH\tCREATED")

	for _, e := range environments {
		composeDir := e.Path
		if ctx.Project.ComposeDir != "" {
			composeDir = filepath.Join(e.Path, ctx.Project.ComposeDir)
		}
		status := docker.GetProjectStatus(composeDir, e.DockerProject)
		age := formatAge(e.CreatedAt)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Name, status, e.Branch, age)
	}

	w.Flush()
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

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tENVIRONMENT\tSTATUS\tBRANCH")

	for _, p := range projects {
		environments, err := ctx.DB.ListEnvironmentsByProject(p.ID)
		if err != nil {
			continue
		}

		if len(environments) == 0 {
			fmt.Fprintf(w, "%s\t(no environments)\t\t\n", p.Name)
			continue
		}

		for _, e := range environments {
			composeDir := e.Path
			if p.ComposeDir != "" {
				composeDir = filepath.Join(e.Path, p.ComposeDir)
			}
			status := docker.GetProjectStatus(composeDir, e.DockerProject)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, e.Name, status, e.Branch)
		}
	}

	w.Flush()
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
