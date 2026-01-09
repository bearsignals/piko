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

var destroyVolumes bool

func init() {
	envCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&destroyVolumes, "volumes", false, "Also remove Docker volumes")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	resolved, err := ResolveEnvironment(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	return operations.DestroyEnvironment(operations.DestroyEnvironmentOptions{
		DB:            resolved.Ctx.DB,
		Project:       resolved.Project,
		Environment:   resolved.Environment,
		RemoveVolumes: destroyVolumes,
		Logger:        &operations.StdoutLogger{},
	})
}
