package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/state"
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

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	dbPath := filepath.Join(cwd, ".piko", "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("not initialized (run 'piko init' first)")
	}
	defer db.Close()

	project, err := db.GetProject()
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	env, err := db.GetEnvironmentByName(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	composeDir := env.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(env.Path, project.ComposeDir)
	}

	var composeCmd *exec.Cmd
	if service != "" {
		composeCmd = exec.Command("docker", "compose", "-p", env.DockerProject, "restart", service)
	} else {
		composeCmd = exec.Command("docker", "compose", "-p", env.DockerProject, "restart")
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
