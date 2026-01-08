package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gwuah/piko/internal/state"
	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show detailed status of an environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

type containerInfo struct {
	Service   string `json:"Service"`
	State     string `json:"State"`
	Status    string `json:"Status"`
	Ports     string `json:"Ports"`
	Name      string `json:"Name"`
	Health    string `json:"Health"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	name := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	dbPath := filepath.Join(cwd, ".piko", "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("not initialized (run 'piko init' first)")
	}
	defer db.Close()

	project, err := db.GetProject()
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	environment, err := db.GetEnvironmentByName(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	composeDir := environment.Path
	if project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, project.ComposeDir)
	}

	relPath, _ := filepath.Rel(cwd, environment.Path)
	if relPath == "" {
		relPath = environment.Path
	}

	sessionName := tmux.SessionName(project.Name, name)
	tmuxStatus := "not running"
	if tmux.SessionExists(sessionName) {
		tmuxStatus = sessionName
	}

	fmt.Printf("Environment: %s\n", environment.Name)
	fmt.Printf("Branch:      %s\n", environment.Branch)
	fmt.Printf("Path:        %s\n", relPath)
	fmt.Printf("Docker:      %s\n", environment.DockerProject)
	fmt.Printf("Tmux:        %s\n", tmuxStatus)

	containers, running, total := getContainerStatus(composeDir, environment.DockerProject)

	if total == 0 {
		fmt.Printf("Status:      stopped (no containers)\n")
	} else if running == total {
		fmt.Printf("Status:      running (%d/%d containers)\n", running, total)
	} else if running == 0 {
		fmt.Printf("Status:      stopped (%d/%d containers)\n", running, total)
	} else {
		fmt.Printf("Status:      partial (%d/%d containers running)\n", running, total)
	}

	if len(containers) > 0 {
		fmt.Println()
		fmt.Printf("%-40s %-10s %s\n", "CONTAINER", "STATUS", "PORTS")
		for _, c := range containers {
			status := c.State
			if c.Health != "" {
				status = fmt.Sprintf("%s (%s)", c.State, c.Health)
			}
			fmt.Printf("%-40s %-10s %s\n", c.Name, status, c.Ports)
		}
	}

	return nil
}

func getContainerStatus(workDir, projectName string) ([]containerInfo, int, int) {
	cmd := exec.Command("docker", "compose", "-p", projectName, "ps", "--format", "json")
	cmd.Dir = workDir

	output, err := cmd.Output()
	if err != nil {
		return nil, 0, 0
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return nil, 0, 0
	}

	var containers []containerInfo
	for _, line := range strings.Split(outputStr, "\n") {
		if line == "" {
			continue
		}
		var c containerInfo
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			continue
		}
		containers = append(containers, c)
	}

	running := 0
	for _, c := range containers {
		if c.State == "running" {
			running++
		}
	}

	return containers, running, len(containers)
}
