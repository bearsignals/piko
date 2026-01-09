package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var projectListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all piko projects",
	RunE:    runProjects,
}

func init() {
	projectCmd.AddCommand(projectListCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
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

	table := NewTable("NAME", "PATH", "ENVIRONMENTS", "CREATED")
	for _, p := range projects {
		environments, err := ctx.DB.ListEnvironmentsByProject(p.ID)
		envCount := 0
		if err == nil {
			envCount = len(environments)
		}
		table.Row(p.Name, p.RootPath, strconv.Itoa(envCount), formatAge(p.CreatedAt))
	}
	table.Flush()
	return nil
}
