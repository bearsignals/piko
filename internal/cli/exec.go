package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/config"
	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <name> <service> [cmd...]",
	Short: "Execute command in a container",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runExec,
}

var shellCmd = &cobra.Command{
	Use:   "shell <name> <service>",
	Short: "Open interactive shell in a container",
	Args:  cobra.ExactArgs(2),
	RunE:  runShell,
}

func init() {
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(shellCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	name := args[0]
	service := args[1]
	var command []string
	if len(args) > 2 {
		command = args[2:]
	}

	return executeInContainer(name, service, command)
}

func runShell(cmd *cobra.Command, args []string) error {
	name := args[0]
	service := args[1]

	return executeInContainer(name, service, nil)
}

func executeInContainer(name, service string, command []string) error {
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

	environment, err := db.GetEnvironmentByName(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	composeDir := environment.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, project.ComposeDir)
	}

	status := docker.GetProjectStatus(composeDir, environment.DockerProject)
	if status != docker.StatusRunning {
		return fmt.Errorf("containers not running (run 'piko up %s' first)", name)
	}

	if len(command) == 0 {
		cfg, _ := config.Load(cwd)
		if cfg != nil && cfg.Shells != nil {
			if shell, ok := cfg.Shells[service]; ok {
				command = []string{"sh", "-c", shell}
			}
		}
		if len(command) == 0 {
			command = []string{"sh"}
		}
	}

	cmdArgs := []string{"compose", "-p", environment.DockerProject, "exec", service}
	cmdArgs = append(cmdArgs, command...)

	dockerCmd := exec.Command("docker", cmdArgs...)
	dockerCmd.Dir = composeDir
	dockerCmd.Stdin = os.Stdin
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
}
