package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open environment worktree in editor",
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

func init() {
	envCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	name := args[0]

	resolved, err := ResolveEnvironmentGlobally(name)
	if err != nil {
		return err
	}
	defer resolved.Close()

	editor := detectEditor()
	fmt.Printf("Opening %s in %s...\n", resolved.Environment.Path, editor)

	editorCmd := exec.Command(editor, resolved.Environment.Path)
	return editorCmd.Start()
}

func detectEditor() string {
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}

	editors := []string{"cursor", "code", "vim", "nano"}
	for _, e := range editors {
		if _, err := exec.LookPath(e); err == nil {
			return e
		}
	}

	return "vim"
}
