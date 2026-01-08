package docker

import (
	"strings"
	"time"

	"github.com/gwuah/piko/internal/run"
)

type ContainerStatus string

const (
	StatusRunning ContainerStatus = "running"
	StatusStopped ContainerStatus = "stopped"
	StatusUnknown ContainerStatus = "unknown"
)

const dockerTimeout = 10 * time.Second

func GetProjectStatus(workDir, projectName string) ContainerStatus {
	output, err := run.Command("docker", "compose", "-p", projectName, "ps", "-q").
		Dir(workDir).
		Timeout(dockerTimeout).
		Output()
	if err != nil {
		return StatusUnknown
	}

	if strings.TrimSpace(string(output)) == "" {
		return StatusStopped
	}

	output, err = run.Command("docker", "compose", "-p", projectName, "ps", "--status", "running", "-q").
		Dir(workDir).
		Timeout(dockerTimeout).
		Output()
	if err != nil {
		return StatusUnknown
	}

	if strings.TrimSpace(string(output)) != "" {
		return StatusRunning
	}

	return StatusStopped
}
