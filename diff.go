package main

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"strings"
)

// DiffResult holds the output of git diff commands.
type DiffResult struct {
	Staged    string `json:"staged"`
	Unstaged  string `json:"unstaged"`
	Untracked string `json:"untracked"`
	Stat      string `json:"stat"`
	Hash      string `json:"hash"`
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

	untracked, err := getUntrackedDiff(repoDir)
	if err != nil {
		// non-fatal: proceed without untracked info
		untracked = ""
	}

	stat, err := runGit(repoDir, "diff", "--stat", "HEAD")
	if err != nil {
		// non-fatal: stat may fail on initial commit
		stat = ""
	}

	combined := staged + unstaged + untracked
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(combined)))

	return &DiffResult{
		Staged:    staged,
		Unstaged:  unstaged,
		Untracked: untracked,
		Stat:      strings.TrimSpace(stat),
		Hash:      hash,
	}, nil
}

// getUntrackedDiff synthesizes a unified diff for each file that is
// present in the working tree but not yet tracked by git. Files matching
// .gitignore are excluded via --exclude-standard.
func getUntrackedDiff(repoDir string) (string, error) {
	// -z emits NUL-separated paths so filenames with spaces/newlines are safe.
	out, err := runGit(repoDir, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", nil
	}

	files := strings.Split(strings.TrimRight(out, "\x00"), "\x00")

	var b strings.Builder
	for _, f := range files {
		if f == "" {
			continue
		}
		// `git diff --no-index /dev/null <file>` produces a standard
		// unified diff with `--- /dev/null` / `+++ b/<file>` headers,
		// which diff2html can render as a new-file addition.
		// `--` disambiguates paths from options; `--no-color` strips ANSI.
		// Exit code 1 (= has differences) is handled by runGit.
		diff, err := runGit(repoDir, "diff", "--no-index", "--no-color", "--", "/dev/null", f)
		if err != nil {
			// Skip unreadable files (symlinks to nowhere, permission errors, etc.)
			// rather than failing the whole diff.
			continue
		}
		b.WriteString(diff)
	}

	return b.String(), nil
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
