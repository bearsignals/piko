package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/env"
	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env <name>",
	Short: "Print PIKO_* environment variables",
	Long:  `Print all PIKO_* environment variables for the given environment. Use with eval: eval $(piko env <name>)`,
	Args:  cobra.ExactArgs(1),
	RunE:  runEnv,
}

var envJSON bool

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.Flags().BoolVar(&envJSON, "json", false, "Output as JSON")
}

func runEnv(cmd *cobra.Command, args []string) error {
	name := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// 1. Open database
	dbPath := filepath.Join(cwd, ".piko", "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("not initialized (run 'piko init' first)")
	}
	defer db.Close()

	// 2. Get project and environment
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

	allocations, err := discoverPorts(environment, composeDir)
	if err != nil {
		return fmt.Errorf("failed to discover ports: %w", err)
	}

	// 4. Build env
	pikoEnv := env.Build(project, environment, allocations)

	// 5. Output
	if envJSON {
		data, err := pikoEnv.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON: %w", err)
		}
		fmt.Println(string(data))
	} else {
		fmt.Println(pikoEnv.ToShellExport())
	}

	return nil
}

func discoverPorts(environment *state.Environment, composeDir string) ([]ports.Allocation, error) {
	var allocations []ports.Allocation

	composeConfig, err := docker.ParseComposeConfig(composeDir)
	if err != nil {
		return nil, err
	}

	for service, containerPorts := range composeConfig.GetServicePorts() {
		for _, containerPort := range containerPorts {
			cmd := exec.Command("docker", "compose",
				"-p", environment.DockerProject,
				"port", service, fmt.Sprintf("%d", containerPort))
			cmd.Dir = composeDir

			output, err := cmd.Output()
			if err != nil {
				continue // Service might not be running
			}

			// Output is like "0.0.0.0:10132" or "[::]:10132"
			outputStr := strings.TrimSpace(string(output))
			parts := strings.Split(outputStr, ":")
			if len(parts) >= 2 {
				hostPort, _ := strconv.Atoi(parts[len(parts)-1])
				if hostPort > 0 {
					allocations = append(allocations, ports.Allocation{
						Service:       service,
						ContainerPort: containerPort,
						HostPort:      hostPort,
					})
				}
			}
		}
	}

	return allocations, nil
}
