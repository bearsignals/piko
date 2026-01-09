package cli

import (
	"github.com/gwuah/piko/internal/operations"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:         "up <name>",
	Short:       "Start containers for an environment",
	Args:        cobra.ExactArgs(1),
	RunE:        runUp,
	Annotations: Requires(ToolDocker),
}

func init() {
	envCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	return operations.UpEnvironment(operations.UpEnvironmentOptions{
		DB:          resolved.Ctx.DB,
		Project:     resolved.Project,
		Environment: resolved.Environment,
		Logger:      &operations.StdoutLogger{},
	})
}
