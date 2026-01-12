package docker

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/run"
)

type ContainerInfo struct {
	Name   string
	State  string
	Health string
}

type PortDiscoveryResult struct {
	Allocations []ports.Allocation
	Containers  []ContainerInfo
	Running     int
	Total       int
}

func DiscoverPorts(composeDir, dockerProject string) (*PortDiscoveryResult, error) {
	result := &PortDiscoveryResult{}

	if dockerProject == "" {
		return result, nil
	}

	output, err := run.Command("docker", "compose", "-p", dockerProject, "ps", "--format", "json").
		Dir(composeDir).
		Timeout(10 * time.Second).
		Output()
	if err != nil {
		return result, nil
	}

	seenPorts := make(map[string]bool)

	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		var c struct {
			Service    string `json:"Service"`
			Name       string `json:"Name"`
			State      string `json:"State"`
			Health     string `json:"Health"`
			Publishers []struct {
				TargetPort    int `json:"TargetPort"`
				PublishedPort int `json:"PublishedPort"`
			} `json:"Publishers"`
		}
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}

		result.Total++
		if c.State == "running" {
			result.Running++
		}

		result.Containers = append(result.Containers, ContainerInfo{
			Name:   c.Name,
			State:  c.State,
			Health: c.Health,
		})

		for _, pub := range c.Publishers {
			if pub.PublishedPort == 0 {
				continue
			}
			key := fmt.Sprintf("%s:%d", c.Service, pub.TargetPort)
			if seenPorts[key] {
				continue
			}
			seenPorts[key] = true

			result.Allocations = append(result.Allocations, ports.Allocation{
				Service:       c.Service,
				ContainerPort: pub.TargetPort,
				HostPort:      pub.PublishedPort,
			})
		}
	}

	return result, nil
}

func IsHTTPPort(port int) bool {
	httpPorts := []int{80, 443, 3000, 3001, 4000, 5000, 5173, 8000, 8080, 8081, 8888, 9000}
	return slices.Contains(httpPorts, port)
}
