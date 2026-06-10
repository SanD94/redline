package cli

import (
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

	commentsData, err := os.ReadFile(filepath.Join(dir, "comments.md"))
	if err != nil {
		t.Fatalf("read comments: %v", err)
	}
	comments := string(commentsData)
	for _, want := range []string{
		"## Comment 4",
		"## Comment 5",
		"## Comment 6",
		"## Comment 7",
		"- **Section:** `discussion`",
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

	for _, rel := range []string{"manifest.json", "comments.md"} {
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
