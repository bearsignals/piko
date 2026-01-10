package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/gwuah/piko/internal/docker"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [name] [service]",
	Short: "Open an environment's service in browser",
	Args:  cobra.RangeArgs(0, 2),
	RunE:  runOpen,
}

func init() {
	envCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name, err := GetEnvNameOrSelect(args)
	if err != nil {
		return err
	}
	var serviceName string
	if len(args) > 1 {
		serviceName = args[1]
	}

	resolved, err := RequireDockerGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	status := docker.GetProjectStatus(resolved.ComposeDir, resolved.Environment.DockerProject)
	if status != docker.StatusRunning {
		return fmt.Errorf("containers not running (run 'piko env up %s' first)", name)
	}

	composeConfig, err := docker.ParseComposeConfig(resolved.ComposeDir)
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
	hostPort, err := getHostPort(resolved.ComposeDir, resolved.Environment.DockerProject, serviceName, containerPort)
	if err != nil {
		return fmt.Errorf("failed to get port for %s: %w", serviceName, err)
	}

	url := fmt.Sprintf("http://localhost:%d", hostPort)
	fmt.Printf("Opening %s in browser...\n", url)

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
