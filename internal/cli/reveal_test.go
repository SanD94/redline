package cli

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestRunRevealSampleDocxPhase1OutputAndDeterminism(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("redline reveal currently needs jj or git for the old-version snapshot")
		}
	}

	root := repoRoot(t)
	sample := filepath.Join(root, "workspace", "sample.docx")
	if _, err := os.Stat(sample); err != nil {
		t.Fatalf("sample fixture missing: %v", err)
	}

	outA := filepath.Join(t.TempDir(), "a")
	outB := filepath.Join(t.TempDir(), "b")

	if err := RunReveal([]string{sample, "--output", outA}); err != nil {
		t.Fatalf("RunReveal(outA) error = %v", err)
	}
	if err := RunReveal([]string{sample, "--output", outB}); err != nil {
		t.Fatalf("RunReveal(outB) error = %v", err)
	}

	assertSampleWorkspace(t, outA)
	assertWorkspaceOutputsEqual(t, outA, outB)
}

func TestRunRevealWritesCommentsReportWhenNoCommentsExist(t *testing.T) {
	if _, err := exec.LookPath("jj"); err != nil {
		if _, err := exec.LookPath("git"); err != nil {
			t.Skip("redline reveal currently needs jj or git for the old-version snapshot")
		}
	}

	docxPath := filepath.Join(t.TempDir(), "no-comments.docx")
	writeCliTestDocx(t, docxPath, map[string]string{
		"word/document.xml": `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>Plain text.</w:t></w:r></w:p></w:body></w:document>`,
		"word/styles.xml":   `<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"/>`,
	})
	out := filepath.Join(t.TempDir(), "out")

	if err := RunReveal([]string{docxPath, "--output", out}); err != nil {
		t.Fatalf("RunReveal() error = %v", err)
	}

	commentsData, err := os.ReadFile(filepath.Join(out, "comments.md"))
	if err != nil {
		t.Fatalf("read comments.md: %v", err)
	}
	if got, want := string(commentsData), "# Comments\n\nNo comments found.\n"; got != want {
		t.Fatalf("comments.md = %q, want %q", got, want)
	}

	if _, err := os.Stat(filepath.Join(out, "source-model.json")); err != nil {
		t.Fatalf("source-model.json missing: %v", err)
	}
}

func assertSampleWorkspace(t *testing.T, dir string) {
	t.Helper()

	manifestPath := filepath.Join(dir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest struct {
		Sections []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Level int    `json:"level"`
		} `json:"sections"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if got, want := len(manifest.Sections), 11; got != want {
		t.Fatalf("section count = %d, want %d", got, want)
	}

	wantIDs := []string{
		"a-very-nice-big-title-for-the-study",
		"abstract",
		"introduction",
		"results",
		"discussion",
		"methods",
		"data-availability",
		"references",
		"acknowledgements",
		"author-contributions",
		"additional-information",
	}
	for i, want := range wantIDs {
		if got := manifest.Sections[i].ID; got != want {
			t.Fatalf("section %d ID = %q, want %q", i, got, want)
		}
	}

	sectionFiles, err := filepath.Glob(filepath.Join(dir, "sections", "*.md"))
	if err != nil {
		t.Fatalf("glob sections: %v", err)
	}
	if got, want := len(sectionFiles), len(wantIDs); got != want {
		t.Fatalf("section file count = %d, want %d", got, want)
	}

	if _, err := os.Stat(filepath.Join(dir, "figures")); err != nil {
		t.Fatalf("figures directory missing: %v", err)
	}
	documentData, err := os.ReadFile(filepath.Join(dir, "document.md"))
	if err != nil {
		t.Fatalf("read document.md: %v", err)
	}
	if !strings.Contains(string(documentData), "# A very nice big title for the study") {
		t.Fatalf("document.md missing first section heading")
	}
	if _, err := os.Stat(filepath.Join(dir, "references.md")); err != nil {
		t.Fatalf("references.md missing: %v", err)
	}

	reviewIntentData, err := os.ReadFile(filepath.Join(dir, "review-intent.json"))
	if err != nil {
		t.Fatalf("read review intent: %v", err)
	}
	var reviewIntent struct {
		Moves []struct {
			Name          string `json:"name"`
			Author        string `json:"author"`
			Date          string `json:"date"`
			Text          string `json:"text"`
			FromSectionID string `json:"fromSectionId"`
			ToSectionID   string `json:"toSectionId"`
			Source        string `json:"source"`
		} `json:"moves"`
	}
	if err := json.Unmarshal(reviewIntentData, &reviewIntent); err != nil {
		t.Fatalf("unmarshal review intent: %v", err)
	}
	if len(reviewIntent.Moves) == 0 {
		t.Fatalf("expected at least one explicit move")
	}
	move := reviewIntent.Moves[0]
	if move.Name == "" {
		t.Fatalf("move name is empty")
	}
	if got, want := move.Author, "Andac, Safa"; got != want {
		t.Fatalf("move author = %q, want %q", got, want)
	}
	if move.Date == "" {
		t.Fatalf("move date is empty")
	}
	if got, want := move.FromSectionID, "methods"; got != want {
		t.Fatalf("move from section = %q, want %q", got, want)
	}
	if got, want := move.ToSectionID, "methods"; got != want {
		t.Fatalf("move to section = %q, want %q", got, want)
	}
	if !strings.Contains(move.Text, "Lorem ipsum dolor sit amet, consectetur adipiscing elit.") {
		t.Fatalf("move text missing expected paragraph start: %q", move.Text)
	}
	if got, want := move.Source, "word/document.xml w:moveFrom/w:moveTo"; got != want {
		t.Fatalf("move source = %q, want %q", got, want)
	}

	sourceModelData, err := os.ReadFile(filepath.Join(dir, "source-model.json"))
	if err != nil {
		t.Fatalf("read source model: %v", err)
	}
	var sourceModel struct {
		DocumentBlocks []struct {
			ID            string `json:"id"`
			Type          string `json:"type"`
			SectionID     string `json:"sectionId"`
			SourcePointer string `json:"sourcePointer"`
			Context       string `json:"context"`
		} `json:"documentBlocks"`
		Comments []struct {
			StableID      string `json:"stableId"`
			AnchorRangeID string `json:"anchorRangeId"`
			SourcePointer string `json:"sourcePointer"`
		} `json:"comments"`
		AnchorRanges []struct {
			ID            string `json:"id"`
			SourcePointer string `json:"sourcePointer"`
		} `json:"anchorRanges"`
	}
	if err := json.Unmarshal(sourceModelData, &sourceModel); err != nil {
		t.Fatalf("unmarshal source model: %v", err)
	}
	if len(sourceModel.DocumentBlocks) == 0 {
		t.Fatalf("source model has no document blocks")
	}
	firstBlock := sourceModel.DocumentBlocks[0]
	if got, want := firstBlock.ID, "block-0001"; got != want {
		t.Fatalf("first source block ID = %q, want %q", got, want)
	}
	if got, want := firstBlock.Type, "paragraph"; got != want {
		t.Fatalf("first source block type = %q, want %q", got, want)
	}
	if !strings.HasPrefix(firstBlock.SourcePointer, "word/document.xml#") {
		t.Fatalf("first source block pointer = %q, want word/document.xml pointer", firstBlock.SourcePointer)
	}
	if len(sourceModel.Comments) == 0 {
		t.Fatalf("source model has no comments")
	}
	if sourceModel.Comments[0].StableID == "" || sourceModel.Comments[0].SourcePointer == "" {
		t.Fatalf("source model comment missing stable identity/source: %#v", sourceModel.Comments[0])
	}
	if len(sourceModel.AnchorRanges) == 0 {
		t.Fatalf("source model has no anchor ranges")
	}
	if sourceModel.AnchorRanges[0].ID == "" || sourceModel.AnchorRanges[0].SourcePointer == "" {
		t.Fatalf("source model anchor missing stable identity/source: %#v", sourceModel.AnchorRanges[0])
	}

	commentsData, err := os.ReadFile(filepath.Join(dir, "comments.md"))
	if err != nil {
		t.Fatalf("read comments: %v", err)
	}
	comments := string(commentsData)
	for _, want := range []string{
		"- **Section:** `discussion`",
		"- **Anchor kind:** normal",
		"- **Anchor kind:** added",
		"- **Anchor kind:** deleted",
		"- **Text:** Nice comment on deleted text.",
		"- **Text:** Nice comment on added text.",
		"- **Text:** Nice comment.",
		"- **Text:** Comment within a comment",
		"- **Text:** Isn’t it?",
		"- **Text:** Done.",
	} {
		if !strings.Contains(comments, want) {
			t.Fatalf("comments.md missing %q in:\n%s", want, comments)
		}
	}

	for _, section := range []string{"results.md", "discussion.md"} {
		data, err := os.ReadFile(filepath.Join(dir, "sections", section))
		if err != nil {
			t.Fatalf("read %s: %v", section, err)
		}
		if len(bytes.TrimSpace(data)) == 0 {
			t.Fatalf("%s is empty", section)
		}
	}
}

func assertWorkspaceOutputsEqual(t *testing.T, a, b string) {
	t.Helper()

	for _, rel := range []string{"manifest.json", "document.md", "comments.md", "review-intent.json", "source-model.json", "summary.json", "summary.md"} {
		assertFilesEqual(t, filepath.Join(a, rel), filepath.Join(b, rel))
	}

	aSections, err := filepath.Glob(filepath.Join(a, "sections", "*.md"))
	if err != nil {
		t.Fatalf("glob a sections: %v", err)
	}
	bSections, err := filepath.Glob(filepath.Join(b, "sections", "*.md"))
	if err != nil {
		t.Fatalf("glob b sections: %v", err)
	}
	sort.Strings(aSections)
	sort.Strings(bSections)
	if len(aSections) != len(bSections) {
		t.Fatalf("section counts differ: %d != %d", len(aSections), len(bSections))
	}
	for i := range aSections {
		if filepath.Base(aSections[i]) != filepath.Base(bSections[i]) {
			t.Fatalf("section file names differ: %s != %s", filepath.Base(aSections[i]), filepath.Base(bSections[i]))
		}
		assertFilesEqual(t, aSections[i], bSections[i])
	}
}

func assertFilesEqual(t *testing.T, a, b string) {
	t.Helper()

	aData, err := os.ReadFile(a)
	if err != nil {
		t.Fatalf("read %s: %v", a, err)
	}
	bData, err := os.ReadFile(b)
	if err != nil {
		t.Fatalf("read %s: %v", b, err)
	}
	if !bytes.Equal(aData, bData) {
		t.Fatalf("files differ: %s != %s", a, b)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

func writeCliTestDocx(t *testing.T, path string, files map[string]string) {
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
