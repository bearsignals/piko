package cli

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/gwuah/piko/internal/config"
	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/env"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Execute the run script for an environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runRun,
}

func init() {
	envCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	resolved, err := RequireDockerGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	cfg, err := config.Load(resolved.Project.RootPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Scripts.Run == "" {
		return fmt.Errorf("no run script defined in .piko.yml (add scripts.run)")
	}

	status := docker.GetProjectStatus(resolved.ComposeDir, resolved.Environment.DockerProject)
	if status != docker.StatusRunning {
		return fmt.Errorf("containers not running (run 'piko env up %s' first)", name)
	}

	portResult, err := docker.DiscoverPorts(resolved.ComposeDir, resolved.Environment.DockerProject)
	if err != nil {
		return fmt.Errorf("failed to discover ports: %w", err)
	}

	pikoEnv := env.Build(resolved.Project, resolved.Environment, portResult.Allocations)
	envVars := append(os.Environ(), pikoEnv.ToEnvSlice()...)

	shellCmd := exec.Command("sh", "-c", cfg.Scripts.Run)
	shellCmd.Dir = resolved.Environment.Path
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

	fmt.Printf("Running scripts.run from .piko.yml...\n")
	return shellCmd.Run()
}
