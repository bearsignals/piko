package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gwuah/piko/internal/config"
	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/env"
	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Execute the run script for an environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runRun,
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	name := args[0]
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

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Scripts.Run == "" {
		return fmt.Errorf("no run script defined in .piko.yml (add scripts.run)")
	}

	composeDir := environment.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, project.ComposeDir)
	}

	status := docker.GetProjectStatus(composeDir, environment.DockerProject)
	if status != docker.StatusRunning {
		return fmt.Errorf("containers not running (run 'piko up %s' first)", name)
	}

	allocations, err := discoverPorts(environment, composeDir)
	if err != nil {
		return fmt.Errorf("failed to discover ports: %w", err)
	}

	pikoEnv := env.Build(project, environment, allocations)
	envVars := append(os.Environ(), pikoEnv.ToEnvSlice()...)

	shellCmd := exec.Command("sh", "-c", cfg.Scripts.Run)
	shellCmd.Dir = environment.Path
	shellCmd.Env = envVars
	shellCmd.Stdin = os.Stdin
	shellCmd.Stdout = os.Stdout
	shellCmd.Stderr = os.Stderr

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		if shellCmd.Process != nil {
			shellCmd.Process.Signal(sig)
		}
	}()

	fmt.Printf("→ Running scripts.run from .piko.yml...\n")
	return shellCmd.Run()
}
