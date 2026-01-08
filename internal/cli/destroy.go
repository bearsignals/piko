package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/config"
	"github.com/gwuah/piko/internal/env"
	"github.com/gwuah/piko/internal/git"
	"github.com/gwuah/piko/internal/ports"
	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var destroyCmd = &cobra.Command{
	Use:   "destroy <name>",
	Short: "Destroy an environment completely",
	Args:  cobra.ExactArgs(1),
	RunE:  runDestroy,
}

var destroyVolumes bool

func init() {
	rootCmd.AddCommand(destroyCmd)
	destroyCmd.Flags().BoolVar(&destroyVolumes, "volumes", false, "Also remove Docker volumes")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	name := args[0]

	ctx, err := NewContext()
	if err != nil {
		return err
	}
	defer ctx.Close()

	environment, err := ctx.GetEnvironment(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	cfg, err := config.Load(ctx.Project.RootPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
		cfg = &config.Config{}
	}

	if cfg.Scripts.Destroy != "" {
		pikoEnv := env.Build(ctx.Project, environment, []ports.Allocation{})
		runner := config.NewScriptRunner(environment.Path, pikoEnv.ToEnvSlice())

		fmt.Println("Running destroy script...")
		if err := runner.RunDestroy(cfg.Scripts.Destroy); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: destroy script failed: %v\n", err)
		}
	}

	sessionName := tmux.SessionName(ctx.Project.Name, name)
	if tmux.SessionExists(sessionName) {
		if err := tmux.KillSession(sessionName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to kill tmux session: %v\n", err)
		} else {
			fmt.Println("✓ Killed tmux session")
		}
	}

	composeDir := environment.Path
	if ctx.Project.ComposeDir != "" {
		composeDir = filepath.Join(environment.Path, ctx.Project.ComposeDir)
	}

	var composeCmd *exec.Cmd
	if destroyVolumes {
		composeCmd = exec.Command("docker", "compose", "-p", environment.DockerProject, "down", "-v")
	} else {
		composeCmd = exec.Command("docker", "compose", "-p", environment.DockerProject, "down")
	}
	composeCmd.Dir = composeDir
	composeCmd.Stdout = os.Stdout
	composeCmd.Stderr = os.Stderr

	if err := composeCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to stop containers: %v\n", err)
	} else {
		fmt.Println("✓ Stopped containers")
		if destroyVolumes {
			fmt.Println("✓ Removed volumes")
		}
	}

	if err := git.RemoveWorktree(environment.Path); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to remove worktree: %v\n", err)
	} else {
		fmt.Println("✓ Removed worktree")
	}

	if err := ctx.DeleteEnvironment(name); err != nil {
		return fmt.Errorf("failed to remove from database: %w", err)
	}
	fmt.Println("✓ Removed from database")

	return nil
}
