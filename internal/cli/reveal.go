package cli

import (
	"fmt"
	"os"

	"github.com/SanD94/redline/internal/docx"
	"github.com/SanD94/redline/internal/wordxml"
	"github.com/SanD94/redline/internal/workspace"
)

type RevealOptions struct {
	InputPath  string
	OutputDir  string
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

	hasComments := "no"
	if r.HasComments() {
		hasComments = "yes"
	}
	fmt.Fprintf(os.Stderr, "  comments present: %s\n", hasComments)

	result, err := wordxml.Parse(r.DocumentXML(), r.CommentsXML(), r.CommentsExtendedXML(), r.StylesXML())
	if err != nil {
		return fmt.Errorf("parse word xml: %w", err)
	}

	fmt.Fprintf(os.Stderr, "  sections: %d\n", len(result.Sections))
	fmt.Fprintf(os.Stderr, "  changes: %d\n", len(result.Changes))
	fmt.Fprintf(os.Stderr, "  comments: %d\n", len(result.Comments))

	wr := workspace.NewWriter(opts.OutputDir)
	if err := wr.Write(result); err != nil {
		return fmt.Errorf("write workspace: %w", err)
	}

	fmt.Fprintf(os.Stderr, "workspace written to: %s\n", opts.OutputDir)
	fmt.Fprintf(os.Stderr, "  sections/    — title-wise markdown files\n")
	fmt.Fprintf(os.Stderr, "  changes.md   — tracked change report\n")
	fmt.Fprintf(os.Stderr, "  comments.md  — comment report\n")
	fmt.Fprintf(os.Stderr, "  review.json  — structured review data\n")

	return nil
}
