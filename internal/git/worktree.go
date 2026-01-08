package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// IsGitRepo checks if the given directory is a git repository.
// It returns true if .git exists (either as a directory or as a file for worktrees).
func IsGitRepo(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	// .git can be a directory (regular repo) or a file (worktree)
	return info.IsDir() || info.Mode().IsRegular()
}

// WorktreeOptions configures worktree creation.
type WorktreeOptions struct {
	Name       string // Name of the worktree (used as directory name)
	BasePath   string // Parent directory for the worktree (e.g., .piko/worktrees)
	BranchName string // Optional: use existing branch instead of creating new
}

// WorktreeResult contains the result of worktree creation.
type WorktreeResult struct {
	Path   string // Full path to the created worktree
	Branch string // Branch name used
}

// CreateWorktree creates a new git worktree.
// If BranchName is empty, creates a new branch with the same name as the worktree.
func CreateWorktree(opts WorktreeOptions) (*WorktreeResult, error) {
	worktreePath := filepath.Join(opts.BasePath, opts.Name)

	var cmd *exec.Cmd
	if opts.BranchName != "" {
		// Use existing branch
		cmd = exec.Command("git", "worktree", "add", worktreePath, opts.BranchName)
	} else {
		// Create new branch with same name as worktree
		cmd = exec.Command("git", "worktree", "add", worktreePath, "-b", opts.Name)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree add failed: %s: %w", string(output), err)
	}

	branch := opts.BranchName
	if branch == "" {
		branch = opts.Name
	}

	return &WorktreeResult{Path: worktreePath, Branch: branch}, nil
}

// BranchExists checks if a branch with the given name exists.
func BranchExists(branchName string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	err := cmd.Run()
	return err == nil, nil
}

// RemoveWorktree removes a git worktree.
func RemoveWorktree(path string) error {
	cmd := exec.Command("git", "worktree", "remove", path, "--force")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s: %w", string(output), err)
	}
	return nil
}
