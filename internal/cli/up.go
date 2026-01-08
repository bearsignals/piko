package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/ports"
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
		fmt.Println("Simple mode environment - no containers to start")
		fmt.Println("Use 'piko attach' to access the tmux session")
		return nil
	}

	composeDir := environment.Path
	if ctx.Project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, ctx.Project.ComposeDir)
	}

	composeConfig, err := docker.ParseComposeConfig(composeDir)
	if err != nil {
		return fmt.Errorf("failed to parse compose config: %w", err)
	}

	servicePorts := composeConfig.GetServicePorts()
	allocations := ports.Allocate(environment.ID, servicePorts)

	composeProject := composeConfig.Project()
	docker.ApplyOverrides(composeProject, ctx.Project.Name, name, allocations)
	pikoComposePath := filepath.Join(composeDir, "docker-compose.piko.yml")
	docker.WriteProjectFile(pikoComposePath, composeProject)

	composeCmd := exec.Command("docker", "compose",
		"-p", environment.DockerProject,
		"-f", "docker-compose.piko.yml",
		"up", "-d")
	composeCmd.Dir = composeDir
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	fmt.Printf("✓ Started containers (%s)\n", environment.DockerProject)
	return nil
}
