package cli

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunAuditReportsNoDifferences(t *testing.T) {
	dir := writeAuditWorkspace(t, "# Introduction\n\nSame.\n", "# Introduction\n\nSame.\n")
	if err := RunAudit([]string{"--workspace", dir}); err != nil {
		t.Fatalf("RunAudit() error = %v", err)
	}
}

func TestRunAuditReportsSectionDifferences(t *testing.T) {
	dir := writeAuditWorkspace(t, "# Introduction\n\nBefore.\n", "# Introduction\n\nAfter.\n")
	err := RunAudit([]string{"--workspace", dir})
	if !errors.Is(err, ErrAuditDifferences) {
		t.Fatalf("RunAudit() error = %v, want ErrAuditDifferences", err)
	}
}

func writeAuditWorkspace(t *testing.T, baseline, current string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".redline", "audit-baseline", "sections"), 0755); err != nil {
		t.Fatalf("mkdir baseline: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sections"), 0755); err != nil {
		t.Fatalf("mkdir sections: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".redline", "audit-baseline", "sections", "introduction.md"), []byte(baseline), 0644); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sections", "introduction.md"), []byte(current), 0644); err != nil {
		t.Fatalf("write current: %v", err)
	}
	return dir
}
