package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	if environment.DockerProject == "" {
		fmt.Println("Simple mode environment - no containers to stop")
		return nil
	}

	composeDir := environment.Path
	if ctx.Project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, ctx.Project.ComposeDir)
	}

	composeCmd := exec.Command("docker", "compose", "-p", environment.DockerProject, "down")
	composeCmd.Dir = composeDir
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	fmt.Println("✓ Stopped containers")
	return nil
}
