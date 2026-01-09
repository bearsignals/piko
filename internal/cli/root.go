package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "piko",
	Short: "Worktree development environments",
	Long:  `piko enables parallel engineering using tmux, worktrees, and containers.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ValidateRequiredTools(cmd)
	},
}

func Execute() error {
	return rootCmd.Execute()
}
