package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gwuah/piko/internal/docker"
	"github.com/gwuah/piko/internal/git"
	"github.com/gwuah/piko/internal/state"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a project for piko",
	Long:  `Initialize the current directory as a piko project. Requires a git repository with a docker-compose file.`,
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// 1. Validate git repo
	if !git.IsGitRepo(cwd) {
		return fmt.Errorf("not a git repository (missing .git)")
	}

	// 2. Detect compose file
	composeFile, err := docker.DetectComposeFile(cwd)
	if err != nil {
		return err
	}
	fmt.Printf("✓ Detected %s\n", composeFile)

	// 3. Check not already initialized
	pikoDir := filepath.Join(cwd, ".piko")
	dbPath := filepath.Join(pikoDir, "state.db")
	if _, err := os.Stat(dbPath); err == nil {
		return fmt.Errorf("already initialized (run 'piko status' to see state)")
	}

	// 4. Create .piko directory
	if err := os.MkdirAll(pikoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .piko: %w", err)
	}

	// 5. Initialize database
	db, err := state.Open(dbPath)
	if err != nil {
		os.RemoveAll(pikoDir) // Clean up on failure
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()

	if err := db.Initialize(); err != nil {
		os.RemoveAll(pikoDir) // Clean up on failure
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	fmt.Println("✓ Created .piko/state.db")

	// 6. Insert project record
	projectName := filepath.Base(cwd)
	project := &state.Project{
		Name:        projectName,
		RootPath:    cwd,
		ComposeFile: composeFile,
	}
	if err := db.InsertProject(project); err != nil {
		os.RemoveAll(pikoDir) // Clean up on failure
		return fmt.Errorf("failed to save project: %w", err)
	}

	// 7. Update .gitignore
	if err := updateGitignore(cwd); err != nil {
		// Non-fatal, just warn
		fmt.Fprintf(os.Stderr, "Warning: could not update .gitignore: %v\n", err)
	}

	fmt.Printf("✓ Project %q initialized\n", projectName)
	return nil
}

func updateGitignore(dir string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	// Check if .piko/ is already in .gitignore
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".piko/" || trimmed == ".piko" {
			return nil // Already present
		}
	}

	// Append .piko/ to .gitignore
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Add newline if file doesn't end with one
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	if _, err := f.WriteString(".piko/\n"); err != nil {
		return err
	}

	return nil
}
