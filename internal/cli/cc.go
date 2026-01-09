package cli

import (
	"github.com/spf13/cobra"
)

var ccCmd = &cobra.Command{
	Use:   "cc",
	Short: "Claude Code integration commands",
	Long:  `Commands for integrating with Claude Code via hooks.`,
}

func init() {
	rootCmd.AddCommand(ccCmd)
}
