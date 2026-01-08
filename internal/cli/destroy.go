package cli

import (
	"fmt"

	"github.com/gwuah/piko/internal/operations"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy <name>",
	Short: "Destroy an environment completely",
	Args:  cobra.ExactArgs(1),
	RunE:  runDestroy,
}

var destroyVolumes bool

func init() {
	rootCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&destroyVolumes, "volumes", false, "Also remove Docker volumes")
}

func runDestroy(cmd *cobra.Command, args []string) error {
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

	return operations.DestroyEnvironment(operations.DestroyEnvironmentOptions{
		DB:            ctx.DB,
		Project:       ctx.Project,
		Environment:   environment,
		RemoveVolumes: destroyVolumes,
		Logger:        &operations.StdoutLogger{},
	})
}
