package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/SanD94/redline/internal/docx"
	"github.com/SanD94/redline/internal/model"
	"github.com/SanD94/redline/internal/wordxml"
	"github.com/SanD94/redline/internal/workspace"
)

type RevealOptions struct {
	InputPath string
	OutputDir string
}

type vcsManager struct {
	dir string
	vcs string // "jj" or "git"
}

func detectVCS(dir string) (*vcsManager, error) {
	if hasDir(filepath.Join(dir, ".jj")) || hasDir(filepath.Join(dir, ".git")) {
		if hasCommand("jj") {
			return &vcsManager{dir: dir, vcs: "jj"}, nil
		}
		return &vcsManager{dir: dir, vcs: "git"}, nil
	}

	if hasCommand("jj") {
		cmd := exec.Command("jj", "git", "init", dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("jj git init: %w\n%s", err, out)
		}
		return &vcsManager{dir: dir, vcs: "jj"}, nil
	}

	if hasCommand("git") {
		cmd := exec.Command("git", "init", dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("git init: %w\n%s", err, out)
		}
		return &vcsManager{dir: dir, vcs: "git"}, nil
	}

	return nil, fmt.Errorf("neither jj nor git found on PATH")
}

func (v *vcsManager) Snapshot(msg string) error {
	switch v.vcs {
	case "jj":
		cmd := exec.Command("jj", "commit", "-m", msg)
		cmd.Dir = v.dir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("jj commit: %w\n%s", err, out)
		}
	case "git":
		add := exec.Command("git", "add", "-A")
		add.Dir = v.dir
		if out, err := add.CombinedOutput(); err != nil {
			return fmt.Errorf("git add: %w\n%s", err, out)
		}
		commit := exec.Command("git", "commit", "-m", msg)
		commit.Dir = v.dir
		if out, err := commit.CombinedOutput(); err != nil {
			return fmt.Errorf("git commit: %w\n%s", err, out)
		}
	}
	return nil
}

func (v *vcsManager) DiffCmd() string {
	switch v.vcs {
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

func RunReveal(args []string) error {
	opts := RevealOptions{
		OutputDir: "./redline-workspace",
	}

	if len(args) < 1 {
		return fmt.Errorf("usage: redline reveal <file.docx> [--output <dir>]")
	}

	opts.InputPath = args[0]

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--output", "-o":
			if i+1 < len(args) {
				i++
				opts.OutputDir = args[i]
			}
		}
	}

	if _, err := os.Stat(opts.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", opts.InputPath)
	}

	fmt.Fprintf(os.Stderr, "reading docx: %s\n", opts.InputPath)
	r, err := docx.Open(opts.InputPath)
	if err != nil {
		return fmt.Errorf("read docx: %w", err)
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("create workspace dir: %w", err)
	}

	vcs, err := detectVCS(opts.OutputDir)
	if err != nil {
		return fmt.Errorf("vcs init: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  vcs: %s\n", vcs.vcs)

	wr := workspace.NewWriter(opts.OutputDir)

	fmt.Fprintf(os.Stderr, "extracting old version (tracked changes rejected)...\n")
	oldResult, err := wordxml.Parse(r.DocumentXML(), r.CommentsXML(), r.CommentsExtendedXML(), r.StylesXML(), model.VersionOld)
	if err != nil {
		return fmt.Errorf("parse old version: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  sections: %d\n", len(oldResult.Sections))

	if err := wr.WriteSections(oldResult); err != nil {
		return fmt.Errorf("write old sections: %w", err)
	}

	os.Remove(filepath.Join(opts.OutputDir, "comments.md"))
	os.Remove(filepath.Join(opts.OutputDir, "review.json"))
	os.Remove(filepath.Join(opts.OutputDir, "manifest.json"))

	fmt.Fprintf(os.Stderr, "saving old version as snapshot...\n")
	if err := vcs.Snapshot("redline: old version snapshot"); err != nil {
		return fmt.Errorf("vcs snapshot: %w", err)
	}

	fmt.Fprintf(os.Stderr, "extracting new version (tracked changes accepted)...\n")
	newResult, err := wordxml.Parse(r.DocumentXML(), r.CommentsXML(), r.CommentsExtendedXML(), r.StylesXML(), model.VersionNew)
	if err != nil {
		return fmt.Errorf("parse new version: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  sections: %d\n", len(newResult.Sections))

	if err := wr.WriteSections(newResult); err != nil {
		return fmt.Errorf("write new sections: %w", err)
	}
	if err := wr.WriteManifest(newResult); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if len(newResult.Comments) > 0 {
		if err := wr.WriteComments(newResult); err != nil {
			return fmt.Errorf("write comments: %w", err)
		}
		if err := wr.WriteJSON(newResult); err != nil {
			return fmt.Errorf("write review json: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "  comments: %d\n", len(newResult.Comments))

	fmt.Fprintf(os.Stderr, "\nworkspace: %s\n", opts.OutputDir)
	fmt.Fprintf(os.Stderr, "  sections/     — title-wise markdown files (new version)\n")
	fmt.Fprintf(os.Stderr, "  manifest.json — section ordering and hierarchy\n")
	if len(newResult.Comments) > 0 {
		fmt.Fprintf(os.Stderr, "  comments.md   — comment report with section references\n")
		fmt.Fprintf(os.Stderr, "  review.json   — structured review data\n")
	}

	diffCmd := vcs.DiffCmd()
	if diffCmd != "" {
		fmt.Fprintf(os.Stderr, "\nchanges from old to new version:\n")
		fmt.Fprintf(os.Stderr, "  cd %s && %s\n", opts.OutputDir, diffCmd)
		if vcs.vcs == "git" {
			fmt.Fprintf(os.Stderr, "\n(review staged diff: cd %s && git diff --cached)\n", opts.OutputDir)
		}
	}

	return nil
}
