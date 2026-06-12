package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RunBatch(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: redline batch <folder> [--output <dir>]")
	}
	inputDir := args[0]
	outputDir := filepath.Join(inputDir, "redline-batch")
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--output", "-o":
			if i+1 >= len(args) {
				return fmt.Errorf("usage: redline batch <folder> [--output <dir>]")
			}
			i++
			outputDir = args[i]
		default:
			return fmt.Errorf("unknown batch option: %s", args[i])
		}
	}
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("read input folder: %w", err)
	}
	successes := 0
	failures := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".docx" || strings.HasPrefix(name, "~$") {
			continue
		}
		inputPath := filepath.Join(inputDir, name)
		workspaceDir := filepath.Join(outputDir, strings.TrimSuffix(name, filepath.Ext(name)))
		if err := RunReveal([]string{inputPath, "--output", workspaceDir}); err != nil {
			failures++
			fmt.Fprintf(os.Stderr, "batch reveal failed for %s: %v\n", inputPath, err)
			continue
		}
		successes++
	}
	fmt.Fprintf(os.Stdout, "batch reveal complete: %d succeeded, %d failed\n", successes, failures)
	if failures > 0 {
		return fmt.Errorf("batch reveal had %d failure(s)", failures)
	}
	return nil
}
