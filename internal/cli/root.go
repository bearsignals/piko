package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "piko",
	Short: "Worktree development environments",
	Long:  `piko enables parallel engineering using tmux, worktrees, and containers.`,
}

func Execute() error {
	return rootCmd.Execute()
}
