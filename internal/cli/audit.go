package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrAuditDifferences = errors.New("audit found markdown differences")

type AuditOptions struct {
	WorkspaceDir string
}

type auditDifference struct {
	Section string
	Before  string
	After   string
}

func RunAudit(args []string) error {
	opts := AuditOptions{WorkspaceDir: "."}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return fmt.Errorf("usage: redline audit [--workspace <dir>]")
			}
			i++
			opts.WorkspaceDir = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return fmt.Errorf("unknown audit option: %s", args[i])
			}
			opts.WorkspaceDir = args[i]
		}
	}

	diffs, err := auditWorkspace(opts.WorkspaceDir)
	if err != nil {
		return err
	}
	if len(diffs) == 0 {
		fmt.Fprintf(os.Stdout, "No content differences found.\n")
		return nil
	}

	fmt.Fprintf(os.Stdout, "Markdown differs from the last revealed accepted Word content.\n\n")
	for _, diff := range diffs {
		fmt.Fprintf(os.Stdout, "## Section `%s`\n", diff.Section)
		fmt.Fprintf(os.Stdout, "- baseline: %s\n", diff.Before)
		fmt.Fprintf(os.Stdout, "- current:  %s\n\n", diff.After)
	}
	fmt.Fprintf(os.Stdout, "Reveal old-to-new review diff remains available separately with your workspace VCS.\n")
	fmt.Fprintf(os.Stdout, "Reproduce source divergence: redline audit --workspace %s\n", opts.WorkspaceDir)
	return ErrAuditDifferences
}

func auditWorkspace(dir string) ([]auditDifference, error) {
	baselineSections := filepath.Join(dir, ".redline", "audit-baseline", "sections")
	currentSections := filepath.Join(dir, "sections")
	entries, err := os.ReadDir(baselineSections)
	if err != nil {
		return nil, fmt.Errorf("read audit baseline: %w", err)
	}

	var diffs []auditDifference
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		basePath := filepath.Join(baselineSections, entry.Name())
		currentPath := filepath.Join(currentSections, entry.Name())
		base, err := os.ReadFile(basePath)
		if err != nil {
			return nil, fmt.Errorf("read baseline %s: %w", entry.Name(), err)
		}
		current, err := os.ReadFile(currentPath)
		if err != nil {
			if os.IsNotExist(err) {
				current = nil
			} else {
				return nil, fmt.Errorf("read current %s: %w", entry.Name(), err)
			}
		}
		if bytes.Equal(base, current) {
			continue
		}
		diffs = append(diffs, auditDifference{
			Section: strings.TrimSuffix(entry.Name(), ".md"),
			Before:  firstContentLine(base),
			After:   firstContentLine(current),
		})
	}

	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Section < diffs[j].Section })
	return diffs, nil
}

func firstContentLine(data []byte) string {
	if len(data) == 0 {
		return "<missing>"
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return "<empty>"
}
