package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "piko",
	Short: "Worktree development environments",
	Long:  `piko creates isolated development environments for each git worktree, orchestrating Docker containers to enable seamless parallel development.`,
}

func Execute() error {
	return rootCmd.Execute()
}
