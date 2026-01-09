package cli

import (
	"github.com/gwuah/piko/internal/operations"
	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new worktree environment",
	Args:  cobra.ExactArgs(1),
	RunE:  runCreate,
}

var (
	createBranch   string
	createNoAttach bool
)

func init() {
	envCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createBranch, "branch", "", "Use existing branch instead of creating new")
	createCmd.Flags().BoolVar(&createNoAttach, "no-attach", false, "Don't attach to tmux session after creation")
}

func runCreate(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	result, err := operations.CreateEnvironment(operations.CreateEnvironmentOptions{
		DB:      ctx.DB,
		Project: ctx.Project,
		Name:    name,
		Branch:  createBranch,
		Logger:  &operations.StdoutLogger{},
	})
	if err != nil {
		return err
	}

	if !createNoAttach && tmux.SessionExists(result.SessionName) {
		return tmux.Attach(result.SessionName)
	}

	return nil
}
