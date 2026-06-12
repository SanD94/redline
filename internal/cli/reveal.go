package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SanD94/redline/internal/docx"
	"github.com/SanD94/redline/internal/model"
	"github.com/SanD94/redline/internal/vcs"
	"github.com/SanD94/redline/internal/wordxml"
	"github.com/SanD94/redline/internal/workspace"
)

type RevealOptions struct {
	InputPath string
	OutputDir string
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

	vcsMgr, err := vcs.Detect(opts.OutputDir)
	if err != nil {
		return fmt.Errorf("vcs init: %w", err)
	}
	fmt.Fprintf(os.Stderr, "  vcs: %s\n", vcsMgr.VCS)

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
	os.Remove(filepath.Join(opts.OutputDir, "document.md"))
	os.Remove(filepath.Join(opts.OutputDir, "manifest.json"))
	os.Remove(filepath.Join(opts.OutputDir, "references.md"))
	os.Remove(filepath.Join(opts.OutputDir, "review-intent.json"))
	os.Remove(filepath.Join(opts.OutputDir, "source-model.json"))

	fmt.Fprintf(os.Stderr, "saving old version as snapshot...\n")
	if err := vcsMgr.Snapshot("redline: old version snapshot"); err != nil {
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
	if err := wr.EnsureAssetDirs(); err != nil {
		return fmt.Errorf("write asset dirs: %w", err)
	}
	if err := wr.WriteDocument(newResult); err != nil {
		return fmt.Errorf("write document: %w", err)
	}
	if err := wr.WriteReferences(newResult); err != nil {
		return fmt.Errorf("write references: %w", err)
	}
	if err := wr.WriteManifest(newResult); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}
	if err := wr.WriteComments(newResult); err != nil {
		return fmt.Errorf("write comments: %w", err)
	}
	if err := wr.WriteReviewIntent(newResult); err != nil {
		return fmt.Errorf("write review intent: %w", err)
	}
	if err := wr.WriteSourceModel(newResult); err != nil {
		return fmt.Errorf("write source model: %w", err)
	}

	fmt.Fprintf(os.Stderr, "  comments: %d\n", len(newResult.Comments))
	fmt.Fprintf(os.Stderr, "  explicit moves: %d\n", len(newResult.Moves))

	fmt.Fprintf(os.Stderr, "\nworkspace: %s\n", opts.OutputDir)
	fmt.Fprintf(os.Stderr, "  sections/     — title-wise markdown files (new version)\n")
	fmt.Fprintf(os.Stderr, "  document.md   — full accepted markdown assembled in section order\n")
	fmt.Fprintf(os.Stderr, "  figures/      — extracted figure assets when available\n")
	fmt.Fprintf(os.Stderr, "  manifest.json — section ordering and hierarchy\n")
	fmt.Fprintf(os.Stderr, "  comments.md   — comment report with section references\n")
	fmt.Fprintf(os.Stderr, "  review-intent.json — explicit Word review actions VCS cannot disambiguate\n")
	fmt.Fprintf(os.Stderr, "  source-model.json — stable comment anchors and DOCX source pointers\n")

	diffCmd := vcsMgr.DiffCmd()
	if diffCmd != "" {
		fmt.Fprintf(os.Stderr, "\nchanges from old to new version:\n")
		fmt.Fprintf(os.Stderr, "  cd %s && %s\n", opts.OutputDir, diffCmd)
		if vcsMgr.VCS == "git" {
			fmt.Fprintf(os.Stderr, "\n(review staged diff: cd %s && git diff --cached)\n", opts.OutputDir)
		}
	}

	return nil
}
