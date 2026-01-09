package cli

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/env"
	"github.com/gwuah/piko/internal/ports"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Environment management commands",
	Long:  `Commands for managing piko environments.`,
}

var varsCmd = &cobra.Command{
	Use:   "vars <name>",
	Short: "Print PIKO_* environment variables",
	Long:  `Print all PIKO_* environment variables for the given environment. Use with eval: eval $(piko env vars <name>)`,
	Args:  cobra.ExactArgs(1),
	RunE:  runVars,
}

var varsJSON bool

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(varsCmd)
	varsCmd.Flags().BoolVar(&varsJSON, "json", false, "Output as JSON")
}

func runVars(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	allocations, err := discoverPorts(resolved.Environment.DockerProject, resolved.ComposeDir)
	if err != nil {
		return fmt.Errorf("failed to discover ports: %w", err)
	}

	pikoEnv := env.Build(resolved.Project, resolved.Environment, allocations)

	if varsJSON {
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

func discoverPorts(dockerProject, composeDir string) ([]ports.Allocation, error) {
	var allocations []ports.Allocation

	if dockerProject == "" {
		return allocations, nil
	}

	composeConfig, err := docker.ParseComposeConfig(composeDir)
	if err != nil {
		return nil, err
	}

	for service, containerPorts := range composeConfig.GetServicePorts() {
		for _, containerPort := range containerPorts {
			cmd := exec.Command("docker", "compose",
				"-p", dockerProject,
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
