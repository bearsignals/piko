package docker

import (
	"fmt"
	"os"

	"github.com/gwuah/piko/internal/ports"
	"gopkg.in/yaml.v3"
)

// OverrideConfig represents the generated docker-compose.piko.yml file.
type OverrideConfig struct {
	Services map[string]OverrideService `yaml:"services,omitempty"`
	Networks map[string]NetworkConfig   `yaml:"networks,omitempty"`
}

// OverrideService represents a service override with port mappings.
type OverrideService struct {
	Ports []string `yaml:"ports,omitempty"`
}

// NetworkConfig represents a network configuration in the override file.
type NetworkConfig struct {
	Name     string `yaml:"name,omitempty"`
	External bool   `yaml:"external,omitempty"`
}

// GenerateOverride creates an override configuration for the given project and environment.
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
		Services: services,
		Networks: map[string]NetworkConfig{
			"default": {Name: networkName},
		},
	}
}

// WriteOverrideFile writes the override configuration to a file.
func WriteOverrideFile(path string, config *OverrideConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal override config: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write override file: %w", err)
	}
	return nil
}
