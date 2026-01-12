package cli

import (
	"github.com/gwuah/piko/internal/operations"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:         "destroy [name]",
	Short:       "Destroy an environment completely",
	Args:        cobra.RangeArgs(0, 1),
	RunE:        runDestroyWithSelection,
	Annotations: Requires(ToolGit, ToolTmux),
}

var (
	keepVolumes  bool
	forceDestroy bool
)

func init() {
	envCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&keepVolumes, "keep-volumes", false, "Keep Docker volumes instead of removing them")
	destroyCmd.Flags().BoolVarP(&forceDestroy, "force", "f", false, "Also delete the git branch")
}

func runDestroyWithSelection(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name, err := GetEnvNameOrSelect(args)
	if err != nil {
		return err
	}
	return runDestroy(name)
}

func runDestroy(name string) error {
	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	api := NewAPIClient()
	if api.IsServerRunning() {
		if err := api.DestroyEnvironment(resolved.Project.ID, resolved.Environment.Name, !keepVolumes, forceDestroy); err == nil {
			return nil
		}
	}

	return operations.DestroyEnvironment(operations.DestroyEnvironmentOptions{
		DB:            resolved.Ctx.DB,
		Project:       resolved.Project,
		Environment:   resolved.Environment,
		RemoveVolumes: !keepVolumes,
		DeleteBranch:  forceDestroy,
		Logger:        &operations.StdoutLogger{},
	})
}
