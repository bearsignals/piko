package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart <name> [service]",
	Short: "Restart containers for an environment",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRestart,
}

func init() {
	envCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	name := args[0]
	var service string
	if len(args) > 1 {
		service = args[1]
	}

	resolved, err := ResolveEnvironment(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	if resolved.Environment.DockerProject == "" {
		fmt.Println("Simple mode environment - no containers to restart")
		return nil
	}

	var composeCmd *exec.Cmd
	if service != "" {
		composeCmd = exec.Command("docker", "compose", "-p", resolved.Environment.DockerProject, "restart", service)
	} else {
		composeCmd = exec.Command("docker", "compose", "-p", resolved.Environment.DockerProject, "restart")
	}

	composeCmd.Dir = resolved.ComposeDir
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to restart containers: %w", err)
	}

	if service != "" {
		fmt.Printf("Restarted %s\n", service)
	} else {
		fmt.Println("Restarted containers")
	}
	return nil
}
