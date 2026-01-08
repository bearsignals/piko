package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <name> [service]",
	Short: "Open an environment's service in browser",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	name := args[0]
	var serviceName string
	if len(args) > 1 {
		serviceName = args[1]
	}

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

	status := docker.GetProjectStatus(composeDir, environment.DockerProject)
	if status != docker.StatusRunning {
		return fmt.Errorf("containers not running (run 'piko up %s' first)", name)
	}

	composeConfig, err := docker.ParseComposeConfig(composeDir)
	if err != nil {
		return fmt.Errorf("failed to parse compose config: %w", err)
	}

	servicePorts := composeConfig.GetServicePorts()

	if serviceName == "" {
		for svc, ports := range servicePorts {
			if len(ports) > 0 {
				serviceName = svc
				break
			}
		}
		if serviceName == "" {
			return fmt.Errorf("no services with exposed ports found")
		}
	}

	ports, ok := servicePorts[serviceName]
	if !ok || len(ports) == 0 {
		return fmt.Errorf("service %q has no exposed ports", serviceName)
	}

	containerPort := ports[0]
	hostPort, err := getHostPort(composeDir, environment.DockerProject, serviceName, containerPort)
	if err != nil {
		return fmt.Errorf("failed to get port for %s: %w", serviceName, err)
	}

	url := fmt.Sprintf("http://localhost:%d", hostPort)
	fmt.Printf("→ Opening %s in browser...\n", url)

	return openBrowser(url)
}

func getHostPort(composeDir, project, service string, containerPort int) (int, error) {
	cmd := exec.Command("docker", "compose",
		"-p", project,
		"port", service, fmt.Sprintf("%d", containerPort))
	cmd.Dir = composeDir

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	outputStr := strings.TrimSpace(string(output))
	parts := strings.Split(outputStr, ":")
	if len(parts) < 2 {
		return 0, fmt.Errorf("unexpected port output: %s", outputStr)
	}

	return strconv.Atoi(parts[len(parts)-1])
}

func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
