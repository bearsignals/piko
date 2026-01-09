package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/gwuah/piko/internal/config"
	"github.com/gwuah/piko/internal/docker"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:         "exec <name> <service> [cmd...]",
	Short:       "Execute command in a container",
	Args:        cobra.MinimumNArgs(2),
	RunE:        runExec,
	Annotations: Requires(ToolDocker),
}

var shellCmd = &cobra.Command{
	Use:         "shell <name> <service>",
	Short:       "Open interactive shell in a container",
	Args:        cobra.ExactArgs(2),
	RunE:        runShell,
	Annotations: Requires(ToolDocker),
}

func init() {
	envCmd.AddCommand(execCmd)
	envCmd.AddCommand(shellCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]
	service := args[1]
	var command []string
	if len(args) > 2 {
		command = args[2:]
	}

	return executeInContainer(name, service, command)
}

func runShell(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]
	service := args[1]

	return executeInContainer(name, service, nil)
}

func executeInContainer(name, service string, command []string) error {
	resolved, err := RequireDockerGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	status := docker.GetProjectStatus(resolved.ComposeDir, resolved.Environment.DockerProject)
	if status != docker.StatusRunning {
		return fmt.Errorf("containers not running (run 'piko env up %s' first)", name)
	}

	if len(command) == 0 {
		cfg, _ := config.Load(resolved.Project.RootPath)
		if cfg != nil && cfg.Shells != nil {
			if shell, ok := cfg.Shells[service]; ok {
				command = []string{"sh", "-c", shell}
			}
		}
		if len(command) == 0 {
			command = []string{"sh"}
		}
	}

	cmdArgs := []string{"compose", "-p", resolved.Environment.DockerProject, "exec", service}
	cmdArgs = append(cmdArgs, command...)

	dockerCmd := exec.Command("docker", cmdArgs...)
	dockerCmd.Dir = resolved.ComposeDir
	dockerCmd.Stdin = os.Stdin
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
}
