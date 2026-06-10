package vcs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Manager struct {
	Dir string
	VCS string
}

func Detect(dir string) (*Manager, error) {
	if hasDir(filepath.Join(dir, ".jj")) || hasDir(filepath.Join(dir, ".git")) {
		if hasCommand("jj") {
			return &Manager{Dir: dir, VCS: "jj"}, nil
		}
		return &Manager{Dir: dir, VCS: "git"}, nil
	}

	if hasCommand("jj") {
		cmd := exec.Command("jj", "git", "init", dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("jj git init: %w\n%s", err, out)
		}
		return &Manager{Dir: dir, VCS: "jj"}, nil
	}

	if hasCommand("git") {
		cmd := exec.Command("git", "init", dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git init: %w\n%s", err, out)
		}
		return &Manager{Dir: dir, VCS: "git"}, nil
	}

	return nil, fmt.Errorf("neither jj nor git found on PATH")
}

func (m *Manager) Snapshot(msg string) error {
	switch m.VCS {
	case "jj":
		cmd := exec.Command("jj", "commit", "-m", msg)
		cmd.Dir = m.Dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("jj commit: %w\n%s", err, out)
		}
	case "git":
		add := exec.Command("git", "add", "-A")
		add.Dir = m.Dir
		if out, err := add.CombinedOutput(); err != nil {
			return fmt.Errorf("git add: %w\n%s", err, out)
		}
		commit := exec.Command(
			"git",
			"-c", "user.name=redline",
			"-c", "user.email=redline@example.invalid",
			"commit", "-m", msg,
		)
		commit.Dir = m.Dir
		if out, err := commit.CombinedOutput(); err != nil {
			return fmt.Errorf("git commit: %w\n%s", err, out)
		}
	}
	return nil
}

func (m *Manager) DiffCmd() string {
	switch m.VCS {
	case "jj":
		return "jj diff"
	case "git":
		return "git diff --no-color"
	}
	return ""
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func hasDir(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
