package docker

import (
	"fmt"
	"os"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/gwuah/piko/internal/ports"
)

func ApplyOverrides(project *types.Project, projectName, envName string, allocations []ports.Allocation) {
	pikoPrefix := fmt.Sprintf("piko-%s-%s", projectName, envName)

	portsByService := make(map[string][]types.ServicePortConfig)
	for _, alloc := range allocations {
		portsByService[alloc.Service] = append(portsByService[alloc.Service], types.ServicePortConfig{
			Target:    uint32(alloc.ContainerPort),
			Published: fmt.Sprintf("%d", alloc.HostPort),
		})
	}

	for name, svc := range project.Services {
		if newPorts, ok := portsByService[name]; ok {
			svc.Ports = newPorts
			project.Services[name] = svc
		}
	}

	project.Networks = types.Networks{
		"default": types.NetworkConfig{
			Name: pikoPrefix,
		},
	}

	newVolumes := types.Volumes{}
	for volName, volConfig := range project.Volumes {
		volConfig.Name = fmt.Sprintf("%s_%s", pikoPrefix, volName)
		newVolumes[volName] = volConfig
	}
	project.Volumes = newVolumes
}

func WriteProjectFile(path string, project *types.Project) error {
	data, err := project.MarshalYAML()
	if err != nil {
		return fmt.Errorf("failed to marshal project: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}
