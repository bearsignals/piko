package git

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gwuah/piko/internal/run"
)

const gitTimeout = 30 * time.Second

func IsGitRepo(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

type WorktreeOptions struct {
	Name       string
	BasePath   string
	BranchName string
	RepoPath   string
}

type WorktreeResult struct {
	Path   string
	Branch string
}

func CreateWorktree(opts WorktreeOptions) (*WorktreeResult, error) {
	worktreePath := filepath.Join(opts.BasePath, opts.Name)

	var cmd *run.Cmd
	if opts.BranchName != "" {
		cmd = run.Command("git", "worktree", "add", worktreePath, "-b", opts.Name, opts.BranchName)
	} else {
		cmd = run.Command("git", "worktree", "add", worktreePath, "-b", opts.Name)
	}

	if opts.RepoPath != "" {
		cmd = cmd.Dir(opts.RepoPath)
	}

	output, err := cmd.Timeout(gitTimeout).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree add failed: %s: %w", string(output), err)
	}

	return &WorktreeResult{Path: worktreePath, Branch: opts.Name}, nil
}

func BranchExists(repoPath, branchName string) (bool, error) {
	err := run.Command("git", "rev-parse", "--verify", branchName).
		Dir(repoPath).
		Timeout(5 * time.Second).
		Run()
	return err == nil, nil
}

func RemoveWorktree(repoPath, worktreePath string) error {
	output, err := run.Command("git", "worktree", "remove", worktreePath, "--force").
		Dir(repoPath).
		Timeout(gitTimeout).
		CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %s: %w", string(output), err)
	}
	return nil
}

func DeleteBranch(repoPath, branchName string) error {
	output, err := run.Command("git", "branch", "-D", branchName).
		Dir(repoPath).
		Timeout(gitTimeout).
		CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch delete failed: %s: %w", string(output), err)
	}
	return nil
}
