package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <name> [service]",
	Short: "View container logs",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runLogs,
}

var (
	logsFollow bool
	logsTail   string
)

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().StringVar(&logsTail, "tail", "", "Number of lines to show from the end")
}

func runLogs(cmd *cobra.Command, args []string) error {
	name := args[0]
	var serviceName string
	if len(args) > 1 {
		serviceName = args[1]
	}

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	environment, err := ctx.GetEnvironment(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	if environment.DockerProject == "" {
		return fmt.Errorf("simple mode environment - no container logs available (use tmux)")
	}

	composeDir := environment.Path
	if ctx.Project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, ctx.Project.ComposeDir)
	}

	cmdArgs := []string{"compose", "-p", environment.DockerProject, "logs"}

	if logsFollow {
		cmdArgs = append(cmdArgs, "-f")
	}

	if logsTail != "" {
		cmdArgs = append(cmdArgs, "--tail", logsTail)
	}

	if serviceName != "" {
		cmdArgs = append(cmdArgs, serviceName)
	}

	dockerCmd := exec.Command("docker", cmdArgs...)
	dockerCmd.Dir = composeDir
	dockerCmd.Stdin = os.Stdin
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
}
