package markdown

import (
	"strings"
	"testing"

	"github.com/SanD94/redline/internal/model"
)

func TestRenderSection(t *testing.T) {
	got := RenderSection(model.Section{
		ID:      "intro",
		Title:   "Introduction",
		Level:   1,
		Content: "First paragraph.",
	})
	want := "# Introduction\n\nFirst paragraph.\n"
	if got != want {
		t.Fatalf("RenderSection() = %q, want %q", got, want)
	}
}

func TestRenderComments(t *testing.T) {
	got := RenderComments(&model.RevealResult{Comments: []model.Comment{
		{ID: 4, Author: "Reviewer", Date: "2026-06-09T13:01:00Z", Text: "Nice comment.", SectionID: "discussion", AnchorKind: "normal", AnchorText: "commented text"},
		{ID: 5, ParentID: 4, Author: "Reviewer", Text: "Reply."},
	}})

	for _, want := range []string{
		"# Comments",
		"## Comment 4",
		"- **Author:** Reviewer",
		"- **Date:** 2026-06-09T13:01:00Z",
		"- **Section:** `discussion`",
		"- **Anchor kind:** normal",
		"- **Anchor text:** commented text",
		"- **Text:** Nice comment.",
		"## Comment 5",
		"- **Parent:** Comment 4",
		"- **Text:** Reply.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderComments() missing %q in:\n%s", want, got)
		}
	}
}
