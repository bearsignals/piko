package ports

import "fmt"

const (
	BasePort             = 10000
	PortRangePerWorktree = 100
)

// Allocation represents a single port allocation for a service.
type Allocation struct {
	Service       string
	ContainerPort int
	HostPort      int
}

func Allocate(worktreeID int64, servicePorts map[string][]int) []Allocation {
	basePort := BasePort + (int(worktreeID) * PortRangePerWorktree)

	var allocations []Allocation
	usedPorts := make(map[int]bool)
	portIndex := 0

	for service, ports := range servicePorts {
		for _, containerPort := range ports {
			hostPort := basePort + (containerPort % 100)
			for usedPorts[hostPort] {
				hostPort = basePort + portIndex
				portIndex++
			}
			usedPorts[hostPort] = true
			allocations = append(allocations, Allocation{
				Service:       service,
				ContainerPort: containerPort,
				HostPort:      hostPort,
			})
		}
	}

	return allocations
}

func (a Allocation) String() string {
	return fmt.Sprintf("%s:%d -> %d", a.Service, a.ContainerPort, a.HostPort)
}

// AllocationsToMap converts allocations to a map of service -> host port (for single port services).
func AllocationsToMap(allocations []Allocation) map[string]int {
	result := make(map[string]int)
	for _, a := range allocations {
		result[a.Service] = a.HostPort
	}
	return result
}
