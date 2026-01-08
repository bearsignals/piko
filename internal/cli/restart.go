package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart <name> [service]",
	Short: "Restart containers for an environment",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRestart,
}

func init() {
	rootCmd.AddCommand(restartCmd)
}

func runRestart(cmd *cobra.Command, args []string) error {
	name := args[0]
	var service string
	if len(args) > 1 {
		service = args[1]
	}

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	environment, err := ctx.GetEnvironment(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	if environment.DockerProject == "" {
		fmt.Println("Simple mode environment - no containers to restart")
		return nil
	}

	composeDir := environment.Path
	if ctx.Project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, ctx.Project.ComposeDir)
	}

	var composeCmd *exec.Cmd
	if service != "" {
		composeCmd = exec.Command("docker", "compose", "-p", environment.DockerProject, "restart", service)
	} else {
		composeCmd = exec.Command("docker", "compose", "-p", environment.DockerProject, "restart")
	}

	composeCmd.Dir = composeDir
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to restart containers: %w", err)
	}

	if service != "" {
		fmt.Printf("✓ Restarted %s\n", service)
	} else {
		fmt.Println("✓ Restarted containers")
	}
	return nil
}
