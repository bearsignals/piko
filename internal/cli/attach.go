package cli

import (
	"fmt"

	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <name>",
	Short: "Attach to an environment's tmux session",
	Long:  "Attach to an environment's tmux session. Use 'project/env' syntax to specify a project explicitly.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttach,
}

func init() {
	envCmd.AddCommand(attachCmd)
}

func runAttach(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	if tmux.IsInsideTmux() {
		return fmt.Errorf("already inside tmux (use 'piko env switch %s' instead)", name)
	}

	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	sessionName := tmux.SessionName(resolved.Project.Name, resolved.Environment.Name)

	if !tmux.SessionExists(sessionName) {
		return fmt.Errorf("session does not exist (run 'piko env up %s' first)", name)
	}

	return tmux.Attach(sessionName)
}
