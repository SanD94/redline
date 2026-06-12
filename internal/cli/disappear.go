package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type DisappearOptions struct {
	WorkspaceDir string
	OutputPath   string
	ReferenceDoc string
	KeepTemp     bool
}

func RunDisappear(args []string) error {
	opts := DisappearOptions{WorkspaceDir: "."}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--output", "-o":
			if i+1 >= len(args) {
				return fmt.Errorf("usage: redline disappear --output <final.docx> [--workspace <dir>] [--reference-doc <docx>]")
			}
			i++
			opts.OutputPath = args[i]
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return fmt.Errorf("usage: redline disappear --output <final.docx> [--workspace <dir>] [--reference-doc <docx>]")
			}
			i++
			opts.WorkspaceDir = args[i]
		case "--reference-doc":
			if i+1 >= len(args) {
				return fmt.Errorf("usage: redline disappear --output <final.docx> [--workspace <dir>] [--reference-doc <docx>]")
			}
			i++
			opts.ReferenceDoc = args[i]
		case "--keep-temp", "--debug":
			opts.KeepTemp = true
		default:
			return fmt.Errorf("unknown disappear option: %s", args[i])
		}
	}
	if opts.OutputPath == "" {
		return fmt.Errorf("disappear requires --output <final.docx>")
	}

	pandoc, err := findPandoc()
	if err != nil {
		return err
	}
	markdown, err := readWorkspaceMarkdownInOrder(opts.WorkspaceDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(opts.OutputPath), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	tmpDir, err := os.MkdirTemp("", "redline-disappear-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	if !opts.KeepTemp {
		defer os.RemoveAll(tmpDir)
	}
	inputPath := filepath.Join(tmpDir, "document.md")
	tmpOutput := filepath.Join(tmpDir, "output.docx")
	if err := os.WriteFile(inputPath, []byte(markdown), 0644); err != nil {
		return fmt.Errorf("write pandoc input: %w", err)
	}

	args = []string{inputPath, "--from", "markdown", "--to", "docx", "--output", tmpOutput, "--resource-path", opts.WorkspaceDir}
	if opts.ReferenceDoc != "" {
		args = append(args, "--reference-doc", opts.ReferenceDoc)
	}
	cmd := exec.Command(pandoc.Path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pandoc docx generation failed: %w", err)
	}
	if err := validateDocx(tmpOutput); err != nil {
		return err
	}
	if err := os.Rename(tmpOutput, opts.OutputPath); err != nil {
		return fmt.Errorf("move generated docx: %w", err)
	}
	if err := writeDisappearMetadata(opts.OutputPath, pandoc); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote clean docx: %s\n", opts.OutputPath)
	if opts.KeepTemp {
		fmt.Fprintf(os.Stderr, "kept temp files: %s\n", tmpDir)
	}
	return nil
}

func readWorkspaceMarkdownInOrder(dir string) (string, error) {
	type sectionEntry struct {
		ID string `json:"id"`
	}
	var manifest struct {
		Sections []sectionEntry `json:"sections"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err == nil {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return "", fmt.Errorf("parse manifest.json: %w", err)
		}
		var parts []string
		for _, sec := range manifest.Sections {
			sectionData, err := os.ReadFile(filepath.Join(dir, "sections", sec.ID+".md"))
			if err != nil {
				return "", fmt.Errorf("read section %s: %w", sec.ID, err)
			}
			parts = append(parts, strings.TrimRight(string(sectionData), "\n"))
		}
		return strings.Join(parts, "\n\n") + "\n", nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read manifest.json: %w", err)
	}
	return readWorkspaceDocument(dir)
}

func writeDisappearMetadata(outputPath string, pandoc PandocInfo) error {
	data, err := json.MarshalIndent(map[string]any{"pandoc": pandoc}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal disappear metadata: %w", err)
	}
	path := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".redline.json"
	return os.WriteFile(path, data, 0644)
}
