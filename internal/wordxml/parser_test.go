package wordxml

import (
	"testing"

	"github.com/SanD94/redline/internal/model"
)

func TestParseTrackedChangesOldAndNewVersions(t *testing.T) {
	documentXML := []byte(`
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Introduction</w:t></w:r></w:p>
    <w:p>
      <w:r><w:t>Before </w:t></w:r>
      <w:del><w:r><w:delText>old</w:delText></w:r></w:del>
      <w:ins><w:r><w:t>new</w:t></w:r></w:ins>
      <w:r><w:t> after.</w:t></w:r>
    </w:p>
  </w:body>
</w:document>`)
	stylesXML := []byte(heading1StylesXML)

	oldResult, err := Parse(documentXML, nil, nil, stylesXML, model.VersionOld)
	if err != nil {
		t.Fatalf("Parse(old) error = %v", err)
	}
	newResult, err := Parse(documentXML, nil, nil, stylesXML, model.VersionNew)
	if err != nil {
		t.Fatalf("Parse(new) error = %v", err)
	}

	if got, want := oldResult.Sections[0].ID, "introduction"; got != want {
		t.Fatalf("old section ID = %q, want %q", got, want)
	}
	if got, want := oldResult.Sections[0].Content, "Before old after."; got != want {
		t.Fatalf("old content = %q, want %q", got, want)
	}
	if got, want := newResult.Sections[0].Content, "Before new after."; got != want {
		t.Fatalf("new content = %q, want %q", got, want)
	}
}

func TestParseCommentsAndThreadLocations(t *testing.T) {
	documentXML := []byte(`
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Discussion</w:t></w:r></w:p>
    <w:p>
      <w:commentRangeStart w:id="1"/>
      <w:r><w:t>Commented text.</w:t></w:r>
      <w:commentRangeEnd w:id="1"/>
      <w:r><w:commentReference w:id="1"/></w:r>
    </w:p>
  </w:body>
</w:document>`)
	commentsXML := []byte(`
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:w14="http://schemas.microsoft.com/office/word/2010/wordml">
  <w:comment w:id="1" w:author="Reviewer" w:date="2026-06-09T13:01:00Z">
    <w:p w14:paraId="parent"><w:r><w:t>Nice comment.</w:t></w:r></w:p>
  </w:comment>
  <w:comment w:id="2" w:author="Reviewer" w:date="2026-06-09T13:02:00Z">
    <w:p w14:paraId="child"><w:r><w:t>Reply.</w:t></w:r></w:p>
  </w:comment>
</w:comments>`)
	commentsExtendedXML := []byte(`
<w15:commentsEx xmlns:w15="http://schemas.microsoft.com/office/word/2012/wordml">
  <w15:commentEx w15:paraId="child" w15:paraIdParent="parent"/>
</w15:commentsEx>`)
	stylesXML := []byte(heading1StylesXML)

	result, err := Parse(documentXML, commentsXML, commentsExtendedXML, stylesXML, model.VersionNew)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got, want := len(result.Comments), 2; got != want {
		t.Fatalf("comment count = %d, want %d", got, want)
	}
	if got, want := result.Comments[0].SectionID, "discussion"; got != want {
		t.Fatalf("comment section = %q, want %q", got, want)
	}
	if got, want := result.Comments[0].Text, "Nice comment."; got != want {
		t.Fatalf("comment text = %q, want %q", got, want)
	}
	if got, want := result.Comments[0].AnchorText, "Commented text."; got != want {
		t.Fatalf("comment anchor text = %q, want %q", got, want)
	}
	if got, want := result.Comments[0].AnchorKind, "normal"; got != want {
		t.Fatalf("comment anchor kind = %q, want %q", got, want)
	}
	if got, want := result.Comments[1].ParentID, 1; got != want {
		t.Fatalf("reply parent ID = %d, want %d", got, want)
	}
}

func TestParseCommentAnchorsForNormalAddedAndDeletedText(t *testing.T) {
	documentXML := []byte(`
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Discussion</w:t></w:r></w:p>
    <w:p>
      <w:r><w:t>Before </w:t></w:r>
      <w:commentRangeStart w:id="1"/>
      <w:r><w:t>normal</w:t></w:r>
      <w:commentRangeEnd w:id="1"/>
      <w:r><w:t> </w:t></w:r>
      <w:ins>
        <w:commentRangeStart w:id="2"/>
        <w:r><w:t>added</w:t></w:r>
        <w:commentRangeEnd w:id="2"/>
      </w:ins>
      <w:r><w:t> </w:t></w:r>
      <w:del>
        <w:commentRangeStart w:id="3"/>
        <w:r><w:delText>deleted</w:delText></w:r>
        <w:commentRangeEnd w:id="3"/>
      </w:del>
      <w:r><w:t> after.</w:t></w:r>
    </w:p>
    <w:p>
      <w:commentRangeStart w:id="4"/>
      <w:r><w:t>mixed normal</w:t></w:r>
      <w:ins><w:r><w:t> and added</w:t></w:r></w:ins>
      <w:commentRangeEnd w:id="4"/>
    </w:p>
  </w:body>
</w:document>`)
	commentsXML := []byte(`
<w:comments xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:comment w:id="1"><w:p><w:r><w:t>Normal comment.</w:t></w:r></w:p></w:comment>
  <w:comment w:id="2"><w:p><w:r><w:t>Added comment.</w:t></w:r></w:p></w:comment>
  <w:comment w:id="3"><w:p><w:r><w:t>Deleted comment.</w:t></w:r></w:p></w:comment>
  <w:comment w:id="4"><w:p><w:r><w:t>Mixed comment.</w:t></w:r></w:p></w:comment>
</w:comments>`)
	stylesXML := []byte(heading1StylesXML)

	result, err := Parse(documentXML, commentsXML, nil, stylesXML, model.VersionNew)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got, want := result.Sections[0].Content, "Before normal added  after.\n\nmixed normal and added"; got != want {
		t.Fatalf("new content = %q, want %q", got, want)
	}
	assertCommentAnchor(t, result.Comments, 1, "normal", "normal")
	assertCommentAnchor(t, result.Comments, 2, "added", "added")
	assertCommentAnchor(t, result.Comments, 3, "deleted", "deleted")
	assertCommentAnchor(t, result.Comments, 4, "mixed", "mixed normal and added")
}

func TestParseMovedTextOldAndNewVersions(t *testing.T) {
	documentXML := []byte(`
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Introduction</w:t></w:r></w:p>
    <w:p>
      <w:moveFromRangeStart w:id="1" w:name="moveA" w:author="Reviewer" w:date="2026-06-09T13:03:00Z"/>
      <w:moveFrom w:id="2">
        <w:r><w:t>Text0</w:t></w:r>
      </w:moveFrom>
      <w:moveFromRangeEnd w:id="1"/>
      <w:r><w:t> another text</w:t></w:r>
    </w:p>
    <w:p>
      <w:r><w:t>another text</w:t></w:r>
      <w:moveToRangeStart w:id="3" w:name="moveA" w:author="Reviewer" w:date="2026-06-09T13:03:00Z"/>
      <w:moveTo w:id="4">
        <w:r><w:t> Text0</w:t></w:r>
      </w:moveTo>
      <w:moveToRangeEnd w:id="1"/>
    </w:p>
  </w:body>
</w:document>`)
	stylesXML := []byte(heading1StylesXML)

	oldResult, err := Parse(documentXML, nil, nil, stylesXML, model.VersionOld)
	if err != nil {
		t.Fatalf("Parse(old) error = %v", err)
	}
	newResult, err := Parse(documentXML, nil, nil, stylesXML, model.VersionNew)
	if err != nil {
		t.Fatalf("Parse(new) error = %v", err)
	}

	// Old version: Text0 from moveFrom at old location, " another text" remains
	if got, want := oldResult.Sections[0].Content, "Text0 another text\n\nanother text"; got != want {
		t.Fatalf("old content = %q, want %q", got, want)
	}
	// New version: moveFrom text gone, " another text" kept; moveTo added at end
	if got, want := newResult.Sections[0].Content, " another text\n\nanother text Text0"; got != want {
		t.Fatalf("new content = %q, want %q", got, want)
	}
	if got, want := len(newResult.Moves), 1; got != want {
		t.Fatalf("move count = %d, want %d", got, want)
	}
	move := newResult.Moves[0]
	if got, want := move.Name, "moveA"; got != want {
		t.Fatalf("move name = %q, want %q", got, want)
	}
	if got, want := move.Author, "Reviewer"; got != want {
		t.Fatalf("move author = %q, want %q", got, want)
	}
	if got, want := move.Date, "2026-06-09T13:03:00Z"; got != want {
		t.Fatalf("move date = %q, want %q", got, want)
	}
	if got, want := move.Text, "Text0"; got != want {
		t.Fatalf("move text = %q, want %q", got, want)
	}
	if got, want := move.FromSectionID, "introduction"; got != want {
		t.Fatalf("move from section = %q, want %q", got, want)
	}
	if got, want := move.ToSectionID, "introduction"; got != want {
		t.Fatalf("move to section = %q, want %q", got, want)
	}
}

func TestParseUsesStylesXMLForHeadingBoundaries(t *testing.T) {
	documentXML := []byte(`
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:pPr><w:pStyle w:val="TitleStyle"/></w:pPr><w:r><w:t>Results</w:t></w:r></w:p>
    <w:p><w:r><w:t>Result body.</w:t></w:r></w:p>
    <w:p><w:pPr><w:pStyle w:val="SubheadingStyle"/></w:pPr><w:r><w:t>Sub result</w:t></w:r></w:p>
  </w:body>
</w:document>`)
	stylesXML := []byte(`
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="TitleStyle"><w:name w:val="heading 1"/></w:style>
  <w:style w:type="paragraph" w:styleId="SubheadingStyle"><w:name w:val="heading 2"/></w:style>
</w:styles>`)

	result, err := Parse(documentXML, nil, nil, stylesXML, model.VersionNew)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got, want := len(result.Sections), 1; got != want {
		t.Fatalf("section count = %d, want %d", got, want)
	}
	if got, want := result.Sections[0].ID, "results"; got != want {
		t.Fatalf("section ID = %q, want %q", got, want)
	}
	if got, want := result.Sections[0].Content, "Result body.\n\n## Sub result"; got != want {
		t.Fatalf("section content = %q, want %q", got, want)
	}
}

const heading1StylesXML = `
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/></w:style>
</w:styles>`

func assertCommentAnchor(t *testing.T, comments []model.Comment, id int, kind, text string) {
	t.Helper()
	for _, cmt := range comments {
		if cmt.ID != id {
			continue
		}
		if got := cmt.SectionID; got != "discussion" {
			t.Fatalf("comment %d section = %q, want discussion", id, got)
		}
		if got := cmt.AnchorKind; got != kind {
			t.Fatalf("comment %d anchor kind = %q, want %q", id, got, kind)
		}
		if got := cmt.AnchorText; got != text {
			t.Fatalf("comment %d anchor text = %q, want %q", id, got, text)
		}
		return
	}
	t.Fatalf("comment %d not found", id)
}
