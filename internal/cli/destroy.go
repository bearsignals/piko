package cli

import (
	"github.com/gwuah/piko/internal/operations"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:         "destroy <name>",
	Short:       "Destroy an environment completely",
	Args:        cobra.ExactArgs(1),
	RunE:        runDestroy,
	Annotations: Requires(ToolGit, ToolTmux),
}

var keepVolumes bool

func init() {
	envCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&keepVolumes, "keep-volumes", false, "Keep Docker volumes instead of removing them")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	api := NewAPIClient()
	if api.IsServerRunning() {
		if err := api.DestroyEnvironment(resolved.Project.ID, resolved.Environment.Name, !keepVolumes); err == nil {
			return nil
		}
	}

	return operations.DestroyEnvironment(operations.DestroyEnvironmentOptions{
		DB:            resolved.Ctx.DB,
		Project:       resolved.Project,
		Environment:   resolved.Environment,
		RemoveVolumes: !keepVolumes,
		Logger:        &operations.StdoutLogger{},
	})
}
