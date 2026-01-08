package cli

import (
	"fmt"

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

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	environment, err := ctx.GetEnvironment(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	return operations.DownEnvironment(operations.DownEnvironmentOptions{
		DB:          ctx.DB,
		Project:     ctx.Project,
		Environment: environment,
		Logger:      &operations.StdoutLogger{},
	})
}
