package cli

import (
	"fmt"

	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <project-name>",
	Short: "Remove a project from piko",
	Long:  `Unregister a project from piko. This does not delete any files, only removes the project from the central database.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRm,
}

var rmForce bool

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force removal even if environments exist")
}

func runRm(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	db, err := state.OpenCentral()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	project, err := db.GetProjectByName(projectName)
	if err != nil {
		return fmt.Errorf("project %q not found", projectName)
	}

	environments, err := db.ListEnvironmentsByProject(project.ID)
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	if len(environments) > 0 && !rmForce {
		return fmt.Errorf("project %q has %d environment(s), use --force to remove anyway or destroy environments first", projectName, len(environments))
	}

	if err := db.DeleteProjectByName(projectName); err != nil {
		return fmt.Errorf("failed to remove project: %w", err)
	}

	fmt.Printf("✓ Removed project %q from piko\n", projectName)
	if len(environments) > 0 {
		fmt.Printf("  Note: %d environment record(s) were also removed from the database\n", len(environments))
		fmt.Println("  Worktree directories and Docker resources were not deleted")
	}

	return nil
}
