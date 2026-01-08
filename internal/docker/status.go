package docker

import (
	"os/exec"
	"strings"
)

type ContainerStatus string

const (
	StatusRunning ContainerStatus = "running"
	StatusStopped ContainerStatus = "stopped"
	StatusUnknown ContainerStatus = "unknown"
)

func GetProjectStatus(workDir, projectName string) ContainerStatus {
	cmd := exec.Command("docker", "compose", "-p", projectName, "ps", "-q")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return StatusUnknown
	}

	if strings.TrimSpace(string(output)) == "" {
		return StatusStopped
	}

	cmd = exec.Command("docker", "compose", "-p", projectName, "ps", "--status", "running", "-q")
	cmd.Dir = workDir

	output, err = cmd.Output()
	if err != nil {
		return StatusUnknown
	}

	if strings.TrimSpace(string(output)) != "" {
		return StatusRunning
	}

	return StatusStopped
}
