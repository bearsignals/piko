package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove the current project from piko",
	Long:  `Unregister the current project from piko. This does not delete any files, only removes the project from the local database.`,
	Args:  cobra.NoArgs,
	RunE:  runRm,
}

var rmForce bool

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().BoolVarP(&rmForce, "force", "f", false, "Force removal even if environments exist")
}

func runRm(cmd *cobra.Command, args []string) error {
	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	environments, err := ctx.ListEnvironments()
	if err != nil {
		return fmt.Errorf("failed to list environments: %w", err)
	}

	if len(environments) > 0 && !rmForce {
		return fmt.Errorf("project %q has %d environment(s), use --force to remove anyway or destroy environments first", ctx.Project.Name, len(environments))
	}

	if err := ctx.DB.DeleteProject(ctx.Project.RootPath); err != nil {
		return fmt.Errorf("failed to remove project: %w", err)
	}

	fmt.Printf("✓ Removed project %q from piko\n", ctx.Project.Name)
	if len(environments) > 0 {
		fmt.Printf("  Note: %d environment record(s) were also removed from the database\n", len(environments))
		fmt.Println("  Worktree directories and Docker resources were not deleted")
	}

	return nil
}
