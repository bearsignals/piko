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
	Use:         "init",
	Short:       "Initialize a project for piko",
	RunE:        runInit,
	Annotations: Requires(ToolGit),
}

var initComposeDir string

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initComposeDir, "compose-dir", "", "Directory containing docker-compose.yml (relative to git root)")
}

func runInit(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if !git.IsGitRepo(cwd) {
		return fmt.Errorf("not a git repository (missing .git)")
	}

	composeSearchDir := cwd
	if initComposeDir != "" {
		composeSearchDir = filepath.Join(cwd, initComposeDir)
		if _, err := os.Stat(composeSearchDir); os.IsNotExist(err) {
			return fmt.Errorf("compose directory does not exist: %s", initComposeDir)
		}
	}

	composeFile, err := docker.DetectComposeFile(composeSearchDir)
	if err != nil {
		composeFile = ""
		fmt.Println("No docker-compose file found, using simple mode")
	} else if initComposeDir != "" {
		fmt.Printf("Detected %s/%s\n", initComposeDir, composeFile)
	} else {
		fmt.Printf("Detected %s\n", composeFile)
	}

	db, err := state.OpenCentral()
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	exists, err := db.ProjectExistsByPath(cwd)
	if err != nil {
		return fmt.Errorf("failed to check project: %w", err)
	}
	if exists {
		return fmt.Errorf("already initialized (run 'piko env list' to see environments)")
	}

	pikoDir := filepath.Join(cwd, ".piko")
	if err := os.MkdirAll(pikoDir, 0755); err != nil {
		return fmt.Errorf("failed to create .piko: %w", err)
	}

	projectName := filepath.Base(cwd)
	project := &state.Project{
		Name:        projectName,
		RootPath:    cwd,
		ComposeFile: composeFile,
		ComposeDir:  initComposeDir,
	}
	if err := db.InsertProject(project); err != nil {
		os.RemoveAll(pikoDir)
		return fmt.Errorf("failed to save project: %w", err)
	}

	if err := updateGitignore(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not update .gitignore: %v\n", err)
	}

	fmt.Printf("Project %q initialized\n", projectName)
	return nil
}

func updateGitignore(dir string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	for line := range strings.SplitSeq(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".piko/" || trimmed == ".piko" {
			return nil
		}
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

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
