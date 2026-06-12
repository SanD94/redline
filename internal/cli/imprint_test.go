package cli

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunImprintPatchesCopyWithoutMutatingSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "received.docx")
	writeCliTestDocx(t, source, map[string]string{
		"[Content_Types].xml": `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"word/document.xml":   `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Original text.</w:t></w:r></w:p></w:body></w:document>`,
	})
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	workspace := filepath.Join(dir, "workspace")
	if err := os.MkdirAll(workspace, 0755); err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "document.md"), []byte("# Introduction\n\nEdited text.\n"), 0644); err != nil {
		t.Fatalf("write document.md: %v", err)
	}
	output := filepath.Join(dir, "patched.docx")

	if err := RunImprint([]string{source, "--workspace", workspace, "--output", output}); err != nil {
		t.Fatalf("RunImprint() error = %v", err)
	}
	after, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source after imprint: %v", err)
	}
	if !bytes.Equal(original, after) {
		t.Fatalf("source docx was mutated")
	}
	if got := readZipEntry(t, output, "word/document.xml"); !strings.Contains(got, "Edited text.") {
		t.Fatalf("patched document missing edited text:\n%s", got)
	}
	if _, err := zip.OpenReader(output); err != nil {
		t.Fatalf("patched docx is not a valid zip: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "patched.warnings.json")); err != nil {
		t.Fatalf("warnings sidecar missing: %v", err)
	}
}

func readZipEntry(t *testing.T, path, name string) string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name != name {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open entry: %v", err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("read entry: %v", err)
		}
		return string(data)
	}
	t.Fatalf("entry %s missing", name)
	return ""
}
