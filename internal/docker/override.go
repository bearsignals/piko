package docker

import (
	"fmt"
	"os"
	"strings"

	"github.com/gwuah/piko/internal/ports"
)

type OverrideConfig struct {
	Services    map[string]OverrideService
	NetworkName string
}

type OverrideService struct {
	Ports []string
}

func GenerateOverride(projectName, envName string, allocations []ports.Allocation) *OverrideConfig {
	networkName := fmt.Sprintf("piko-%s-%s", projectName, envName)

	services := make(map[string]OverrideService)
	for _, alloc := range allocations {
		svc, exists := services[alloc.Service]
		if !exists {
			svc = OverrideService{Ports: []string{}}
		}
		svc.Ports = append(svc.Ports, fmt.Sprintf("%d:%d", alloc.HostPort, alloc.ContainerPort))
		services[alloc.Service] = svc
	}

	return &OverrideConfig{
		Services:    services,
		NetworkName: networkName,
	}
}

func WriteOverrideFile(path string, config *OverrideConfig) error {
	var sb strings.Builder

	sb.WriteString("services:\n")
	for name, svc := range config.Services {
		sb.WriteString(fmt.Sprintf("  %s:\n", name))
		sb.WriteString("    ports: !override\n")
		for _, port := range svc.Ports {
			sb.WriteString(fmt.Sprintf("      - \"%s\"\n", port))
		}
	}

	sb.WriteString("\nnetworks:\n")
	sb.WriteString("  default:\n")
	sb.WriteString(fmt.Sprintf("    name: %s\n", config.NetworkName))

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write override file: %w", err)
	}
	return nil
}
