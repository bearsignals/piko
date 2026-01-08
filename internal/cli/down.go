package cli

import (
	"github.com/gwuah/piko/internal/operations"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down <name>",
	Short: "Stop containers for an environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runDown,
}

func init() {
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	name := args[0]

	resolved, err := ResolveEnvironment(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	return operations.DownEnvironment(operations.DownEnvironmentOptions{
		DB:          resolved.Ctx.DB,
		Project:     resolved.Project,
		Environment: resolved.Environment,
		Logger:      &operations.StdoutLogger{},
	})
}
