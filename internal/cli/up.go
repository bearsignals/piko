package cli

import (
	"fmt"

	"github.com/gwuah/piko/internal/operations"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up <name>",
	Short: "Start containers for an environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
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

	return operations.UpEnvironment(operations.UpEnvironmentOptions{
		DB:          ctx.DB,
		Project:     ctx.Project,
		Environment: environment,
		Logger:      &operations.StdoutLogger{},
	})
}
