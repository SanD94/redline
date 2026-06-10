package docx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenReadsCoreWordParts(t *testing.T) {
	docxPath := filepath.Join(t.TempDir(), "sample.docx")
	writeTestDocx(t, docxPath, map[string]string{
		"word/document.xml":         `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body/></w:document>`,
		"word/comments.xml":         `<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`,
		"word/commentsExtended.xml": `<w15:commentsEx xmlns:w15="http://schemas.microsoft.com/office/word/2012/wordml"/>`,
		"word/styles.xml":           `<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`,
	})

	r, err := Open(docxPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if len(r.DocumentXML()) == 0 {
		t.Fatal("DocumentXML() is empty")
	}
	if !r.HasComments() {
		t.Fatal("HasComments() = false, want true")
	}
	if len(r.CommentsExtendedXML()) == 0 {
		t.Fatal("CommentsExtendedXML() is empty")
	}
	if len(r.StylesXML()) == 0 {
		t.Fatal("StylesXML() is empty")
	}
}

func TestOpenRequiresDocumentXML(t *testing.T) {
	docxPath := filepath.Join(t.TempDir(), "broken.docx")
	writeTestDocx(t, docxPath, map[string]string{
		"word/comments.xml": `<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`,
	})

	if _, err := Open(docxPath); err == nil {
		t.Fatal("Open() error = nil, want missing document.xml error")
	}
}

func writeTestDocx(t *testing.T, path string, files map[string]string) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create test docx: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
}
