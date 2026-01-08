package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

var composeFilenames = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// DetectComposeFile finds the compose file in the given directory.
// Returns the filename (not full path) if found.
func DetectComposeFile(dir string) (string, error) {
	for _, name := range composeFilenames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("no compose file found (tried: %v)", composeFilenames)
}

// ComposeConfig represents the parsed docker compose configuration.
type ComposeConfig struct {
	Services map[string]ServiceConfig `json:"services"`
	Networks map[string]interface{}   `json:"networks"`
	Volumes  map[string]interface{}   `json:"volumes"`
}

// ServiceConfig represents a service in the compose file.
type ServiceConfig struct {
	Ports []PortMapping `json:"ports"`
}

// PortMapping represents a port mapping in a service.
type PortMapping struct {
	Target    int    `json:"target"`    // Container port
	Published string `json:"published"` // Host port (may be empty or range)
	Protocol  string `json:"protocol"`  // tcp/udp
}

// ParseComposeConfig runs `docker compose config --format json` and parses the output.
func ParseComposeConfig(workDir string) (*ComposeConfig, error) {
	cmd := exec.Command("docker", "compose", "config", "--format", "json")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("docker compose config failed: %s: %w", string(exitErr.Stderr), err)
		}
		return nil, fmt.Errorf("docker compose config failed: %w", err)
	}

	var config ComposeConfig
	if err := json.Unmarshal(output, &config); err != nil {
		return nil, fmt.Errorf("failed to parse compose config: %w", err)
	}

	return &config, nil
}

// GetServicePorts returns a map of service names to their container ports.
func (c *ComposeConfig) GetServicePorts() map[string][]int {
	result := make(map[string][]int)
	for name, svc := range c.Services {
		var ports []int
		for _, p := range svc.Ports {
			if p.Target > 0 {
				ports = append(ports, p.Target)
			}
		}
		if len(ports) > 0 {
			result[name] = ports
		}
	}
	return result
}

// GetServiceNames returns a list of all service names.
func (c *ComposeConfig) GetServiceNames() []string {
	names := make([]string, 0, len(c.Services))
	for name := range c.Services {
		names = append(names, name)
	}
	return names
}
