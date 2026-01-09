package cli

import (
	"github.com/gwuah/piko/internal/operations"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:         "down <name>",
	Short:       "Stop containers for an environment",
	Args:        cobra.ExactArgs(1),
	RunE:        runDown,
	Annotations: Requires(ToolDocker),
}

func init() {
	envCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	api := NewAPIClient()
	if api.IsServerRunning() {
		if err := api.Down(resolved.Project.ID, resolved.Environment.Name); err == nil {
			return nil
		}
	}

	return operations.DownEnvironment(operations.DownEnvironmentOptions{
		DB:          resolved.Ctx.DB,
		Project:     resolved.Project,
		Environment: resolved.Environment,
		Logger:      &operations.StdoutLogger{},
	})
}
