package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunDisappearUsesPandocAndWritesDocx(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fake pandoc shell script is POSIX-specific")
	}
	dir := t.TempDir()
	fakePandoc := filepath.Join(dir, "pandoc")
	if err := os.WriteFile(fakePandoc, []byte(`#!/bin/sh
if [ "$1" = "--version" ]; then
  echo "pandoc 3.0-test"
  exit 0
fi
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--output" ]; then out="$arg"; fi
  prev="$arg"
done
python3 - "$out" <<'PY'
import sys, zipfile
with zipfile.ZipFile(sys.argv[1], 'w') as z:
    z.writestr('[Content_Types].xml', '<?xml version="1.0"?><Types/>')
    z.writestr('word/document.xml', '<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body/></w:document>')
PY
`), 0755); err != nil {
		t.Fatalf("write fake pandoc: %v", err)
	}
	t.Setenv("REDLINE_PANDOC", fakePandoc)

	workspace := filepath.Join(dir, "workspace")
	if err := os.MkdirAll(filepath.Join(workspace, "sections"), 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "manifest.json"), []byte(`{"sections":[{"id":"intro"},{"id":"methods"}]}`), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "sections", "intro.md"), []byte("# Intro\n\nA.\n"), 0644); err != nil {
		t.Fatalf("write intro: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "sections", "methods.md"), []byte("# Methods\n\nB.\n"), 0644); err != nil {
		t.Fatalf("write methods: %v", err)
	}
	output := filepath.Join(dir, "final.docx")
	if err := RunDisappear([]string{"--workspace", workspace, "--output", output}); err != nil {
		t.Fatalf("RunDisappear() error = %v", err)
	}
	if err := validateDocx(output); err != nil {
		t.Fatalf("generated docx invalid: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "final.redline.json")); err != nil {
		t.Fatalf("metadata missing: %v", err)
	}
}
