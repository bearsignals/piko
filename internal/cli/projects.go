package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:     "projects",
	Aliases: []string{"ps"},
	Short:   "List all piko projects",
	RunE:    runProjects,
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
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
	fmt.Fprintln(w, "NAME\tPATH\tENVIRONMENTS\tCREATED")

	for _, p := range projects {
		environments, err := ctx.DB.ListEnvironmentsByProject(p.ID)
		envCount := 0
		if err == nil {
			envCount = len(environments)
		}
		age := formatAge(p.CreatedAt)
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", p.Name, p.RootPath, envCount, age)
	}

	w.Flush()
	return nil
}
