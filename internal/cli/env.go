package cli

import (
	"fmt"
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

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	environment, err := ctx.GetEnvironment(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	composeDir := environment.Path
	if ctx.Project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, ctx.Project.ComposeDir)
	}

	allocations, err := discoverPorts(environment, composeDir)
	if err != nil {
		return fmt.Errorf("failed to discover ports: %w", err)
	}

	pikoEnv := env.Build(ctx.Project, environment, allocations)

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
				continue
			}

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
