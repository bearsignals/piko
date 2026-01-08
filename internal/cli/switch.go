package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gwuah/piko/internal/state"
	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch <name>",
	Short: "Switch to an environment's tmux session",
	Args:  cobra.ExactArgs(1),
	RunE:  runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	name := args[0]

	if !tmux.IsInsideTmux() {
		return fmt.Errorf("not inside tmux (use 'piko attach %s' instead)", name)
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

	_, err = db.GetEnvironmentByName(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	sessionName := tmux.SessionName(project.Name, name)

	if !tmux.SessionExists(sessionName) {
		return fmt.Errorf("session does not exist (run 'piko up %s' first)", name)
	}

	return tmux.Switch(sessionName)
}
