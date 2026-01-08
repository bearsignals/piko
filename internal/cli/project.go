package cli

import (
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project management commands",
	Long:  `Commands for managing piko projects.`,
}

func init() {
	rootCmd.AddCommand(projectCmd)
}
