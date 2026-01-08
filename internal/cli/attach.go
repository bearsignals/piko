package cli

import (
	"fmt"
	"strings"

	"github.com/gwuah/piko/internal/state"
	"github.com/gwuah/piko/internal/tmux"
	"github.com/spf13/cobra"
)

var attachCmd = &cobra.Command{
	Use:   "attach <name>",
	Short: "Attach to an environment's tmux session",
	Long:  "Attach to an environment's tmux session. Use 'project/env' syntax to specify a project explicitly.",
	Args:  cobra.ExactArgs(1),
	RunE:  runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
}

func runAttach(cmd *cobra.Command, args []string) error {
	name := args[0]

	if tmux.IsInsideTmux() {
		return fmt.Errorf("already inside tmux (use 'piko switch %s' instead)", name)
	}

	ctx, err := NewContextWithoutProject()
	if err != nil {
		return err
	}
	defer ctx.Close()

	var projectName, envName string
	var project *state.Project

	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		projectName, envName = parts[0], parts[1]
		project, err = ctx.DB.GetProjectByName(projectName)
		if err != nil {
			return fmt.Errorf("project %q not found", projectName)
		}
		_, err = ctx.DB.GetEnvironmentByName(project.ID, envName)
		if err != nil {
			return fmt.Errorf("environment %q not found in project %q", envName, projectName)
		}
	} else {
		envName = name
		results, err := ctx.DB.FindEnvironmentGlobally(envName)
		if err != nil {
			return err
		}
		if len(results) == 0 {
			return fmt.Errorf("environment %q not found", envName)
		}
		if len(results) > 1 {
			fmt.Printf("Multiple environments named %q found:\n", envName)
			for _, r := range results {
				fmt.Printf("  %s/%s\n", r.Project.Name, r.Environment.Name)
			}
			return fmt.Errorf("use 'piko attach <project>/%s' to specify which one", envName)
		}
		project = results[0].Project
		projectName = project.Name
	}

	sessionName := tmux.SessionName(projectName, envName)

	if !tmux.SessionExists(sessionName) {
		return fmt.Errorf("session does not exist (run 'piko up %s' first)", name)
	}

	return tmux.Attach(sessionName)
}
