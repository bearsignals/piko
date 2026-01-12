package cli

import (
	"github.com/spf13/cobra"
)

var rootCreateCmd = &cobra.Command{
	Use:         "create <name>",
	Short:       "Create a new worktree environment (alias for 'piko env create')",
	Long:        "Create a new worktree environment. Use project/name syntax to create in a specific project from anywhere.",
	Args:        cobra.ExactArgs(1),
	RunE:        runCreate,
	Annotations: Requires(ToolGit, ToolTmux),
}

var rootDestroyCmd = &cobra.Command{
	Use:         "destroy [name]",
	Short:       "Destroy an environment completely (alias for 'piko env destroy')",
	Args:        cobra.RangeArgs(0, 1),
	RunE:        runDestroyWithSelection,
	Annotations: Requires(ToolGit, ToolTmux),
}

var rootSwitchCmd = &cobra.Command{
	Use:         "switch [name]",
	Short:       "Switch to an environment's tmux session (alias for 'piko env switch')",
	Long:        "Switch to an environment's tmux session. Use 'project/env' syntax to specify a project explicitly.",
	Args:        cobra.RangeArgs(0, 1),
	RunE:        runSwitch,
	Annotations: Requires(ToolTmux),
}

var rootPickCmd = &cobra.Command{
	Use:   "pick",
	Short: "Fuzzy pick an environment to attach/switch to (alias for 'piko env pick')",
	RunE:  runPick,
}

func init() {
	rootCmd.AddCommand(rootCreateCmd)
	rootCmd.AddCommand(rootDestroyCmd)
	rootCmd.AddCommand(rootSwitchCmd)
	rootCmd.AddCommand(rootPickCmd)
	rootCreateCmd.Flags().StringVar(&createBranch, "branch", "", "Base branch to create the new branch from")
	rootCreateCmd.Flags().BoolVar(&createNoAttach, "no-attach", false, "Don't attach to tmux session after creation")
	rootDestroyCmd.Flags().BoolVar(&keepVolumes, "keep-volumes", false, "Keep Docker volumes instead of removing them")
	rootDestroyCmd.Flags().BoolVarP(&forceDestroy, "force", "f", false, "Also delete the git branch")
}
