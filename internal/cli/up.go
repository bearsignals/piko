package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up <name>",
	Short: "Start containers for an environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runUp,
}

func init() {
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
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

	env, err := db.GetEnvironmentByName(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	composeDir := env.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(env.Path, project.ComposeDir)
	}

	composeConfig, err := docker.ParseComposeConfig(composeDir)
	if err != nil {
		return fmt.Errorf("failed to parse compose config: %w", err)
	}

	servicePorts := composeConfig.GetServicePorts()
	allocations := ports.Allocate(env.ID, servicePorts)

	override := docker.GenerateOverride(project.Name, name, allocations)
	overridePath := filepath.Join(composeDir, "docker-compose.piko.yml")
	if err := docker.WriteOverrideFile(overridePath, override); err != nil {
		return fmt.Errorf("failed to write override file: %w", err)
	}

	composeCmd := exec.Command("docker", "compose",
		"-p", env.DockerProject,
		"-f", "docker-compose.yml",
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	composeCmd.Dir = composeDir
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	fmt.Println("✓ Started containers")
	return nil
}
