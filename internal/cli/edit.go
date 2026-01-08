package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Open environment worktree in editor",
	Args:  cobra.ExactArgs(1),
	RunE:  runEdit,
}

func init() {
	rootCmd.AddCommand(editCmd)
}

func runEdit(cmd *cobra.Command, args []string) error {
	name := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	dbPath := filepath.Join(cwd, ".piko", "state.db")
	db, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("not initialized (run 'piko init' first)")
	}
	defer db.Close()

	environment, err := db.GetEnvironmentByName(name)
	if err != nil {
		return fmt.Errorf("environment %q not found", name)
	}

	editor := detectEditor()
	fmt.Printf("→ Opening %s in %s...\n", environment.Path, editor)

	editorCmd := exec.Command(editor, environment.Path)
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
