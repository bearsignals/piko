package cli

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:         "logs [name] [service]",
	Short:       "View container logs",
	Args:        cobra.RangeArgs(0, 2),
	RunE:        runLogs,
	Annotations: Requires(ToolDocker),
}

var (
	logsFollow bool
	logsTail   string
)

func init() {
	envCmd.AddCommand(logsCmd)
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().StringVar(&logsTail, "tail", "", "Number of lines to show from the end")
}

func runLogs(cmd *cobra.Command, args []string) error {
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

	cmdArgs := []string{"compose", "-p", resolved.Environment.DockerProject, "logs"}

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
	dockerCmd.Dir = resolved.ComposeDir
	dockerCmd.Stdin = os.Stdin
	dockerCmd.Stdout = os.Stdout
	dockerCmd.Stderr = os.Stderr

	return dockerCmd.Run()
}
