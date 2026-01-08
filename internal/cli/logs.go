package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/state"
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
