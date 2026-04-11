package main

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"strings"
)

// DiffResult holds the output of git diff commands.
type DiffResult struct {
	Staged   string `json:"staged"`
	Unstaged string `json:"unstaged"`
	Stat     string `json:"stat"`
	Hash     string `json:"hash"`
}

// GetDiff runs git diff commands in the given repository directory
// and returns the combined result.
func GetDiff(repoDir string) (*DiffResult, error) {
	staged, err := runGit(repoDir, "diff", "--cached")
	if err != nil {
		return nil, fmt.Errorf("git diff --cached: %w", err)
	}

	unstaged, err := runGit(repoDir, "diff")
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}

	stat, err := runGit(repoDir, "diff", "--stat", "HEAD")
	if err != nil {
		// non-fatal: stat may fail on initial commit
		stat = ""
	}

	combined := staged + unstaged
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(combined)))

	return &DiffResult{
		Staged:   staged,
		Unstaged: unstaged,
		Stat:     strings.TrimSpace(stat),
		Hash:     hash,
	}, nil
}

// GetRepoName returns the basename of the git repository root.
func GetRepoName(repoDir string) string {
	parts := strings.Split(strings.TrimRight(repoDir, "/"), "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

// GetRepoRoot returns the top-level directory of the git repository.
func GetRepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// git diff returns exit code 1 when there are differences
			if exitErr.ExitCode() == 1 {
				return string(out), nil
			}
		}
		return "", err
	}
	return string(out), nil
}
