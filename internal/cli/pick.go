package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var pickCmd = &cobra.Command{
	Use:   "pick",
	Short: "Fuzzy pick an environment to attach/switch to",
	RunE:  runPick,
}

func init() {
	envCmd.AddCommand(pickCmd)
}

func runPick(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	if _, err := exec.LookPath("fzf"); err != nil {
		return fmt.Errorf("fzf not found (install with: brew install fzf)")
	}

	sessions, err := tmux.ListPikoSessions()
	if err != nil {
		return fmt.Errorf("failed to list sessions: %w", err)
	}

	if len(sessions) == 0 {
		return fmt.Errorf("no piko sessions found")
	}

	fzf := exec.Command("fzf", "--height=40%", "--reverse", "--prompt=piko> ")
	fzf.Stdin = strings.NewReader(strings.Join(sessions, "\n"))
	fzf.Stderr = os.Stderr

	output, err := fzf.Output()
	if err != nil {
		return nil
	}

	selected := strings.TrimSpace(string(output))
	if selected == "" {
		return nil
	}

	if tmux.IsInsideTmux() {
		return tmux.Switch(selected)
	}
	return tmux.Attach(selected)
}
