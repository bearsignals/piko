package cli

import (
	"fmt"

	"github.com/gwuah/piko/internal/operations"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:         "restart <name> [service]",
	Short:       "Restart containers for an environment",
	Args:        cobra.RangeArgs(1, 2),
	RunE:        runRestart,
	Annotations: Requires(ToolDocker),
}

func init() {
	envCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]
	var service string
	if len(args) > 1 {
		service = args[1]
	}

	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	if resolved.Environment.DockerProject == "" {
		fmt.Println("Simple mode environment - no containers to restart")
		return nil
	}

	api := NewAPIClient()
	if api.IsServerRunning() {
		if err := api.Restart(resolved.Project.ID, resolved.Environment.Name, service); err == nil {
			if service != "" {
				fmt.Printf("Restarted %s\n", service)
			} else {
				fmt.Println("Restarted containers")
			}
			return nil
		}
	}

	return operations.RestartEnvironment(operations.RestartEnvironmentOptions{
		DB:          resolved.Ctx.DB,
		Project:     resolved.Project,
		Environment: resolved.Environment,
		Service:     service,
		Logger:      &operations.StdoutLogger{},
	})
}
