package cli

import (
	"fmt"

	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <name>",
	Short: "Attach to an environment's tmux session",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
}

func runAttach(cmd *cobra.Command, args []string) error {
	name := args[0]

	if tmux.IsInsideTmux() {
		return fmt.Errorf("already inside tmux (use 'piko switch %s' instead)", name)
	}

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	_, err = ctx.GetEnvironment(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	sessionName := tmux.SessionName(ctx.Project.Name, name)

	if !tmux.SessionExists(sessionName) {
		return fmt.Errorf("session does not exist (run 'piko up %s' first)", name)
	}

	return tmux.Attach(sessionName)
}
