package markdown

import (
	"fmt"
	"strings"

	"github.com/SanD94/redline/internal/model"
)

func RenderSection(sec model.Section) string {
	var b strings.Builder

	if sec.Title != "" {
		prefix := strings.Repeat("#", sec.Level)
		b.WriteString(fmt.Sprintf("%s %s\n\n", prefix, sec.Title))
	}

	b.WriteString(sec.Content)
	b.WriteString("\n")

	return b.String()
}

func RenderChanges(result *model.RevealResult) string {
	var b strings.Builder
	b.WriteString("# Changes\n\n")

	if len(result.Changes) == 0 {
		b.WriteString("No tracked changes found.\n")
		return b.String()
	}

	for _, ch := range result.Changes {
		badge := "++"
		if ch.Type == model.ChangeDeletion {
			badge = "--"
		}
		b.WriteString(fmt.Sprintf("## Change %d (%s)\n\n", ch.ID, string(ch.Type)))
		if ch.Author != "" {
			b.WriteString(fmt.Sprintf("- **Author:** %s\n", ch.Author))
		}
		if ch.Date != "" {
			b.WriteString(fmt.Sprintf("- **Date:** %s\n", ch.Date))
		}
		if ch.SectionID != "" {
			b.WriteString(fmt.Sprintf("- **Section:** `%s`\n", ch.SectionID))
		}
		b.WriteString(fmt.Sprintf("- **Text:** `[%s%s%s]`\n\n", badge, ch.Text, badge))
	}

	return b.String()
}

func RenderComments(result *model.RevealResult) string {
	var b strings.Builder
	b.WriteString("# Comments\n\n")

	if len(result.Comments) == 0 {
		b.WriteString("No comments found.\n")
		return b.String()
	}

	for _, cmt := range result.Comments {
		b.WriteString(fmt.Sprintf("## Comment %d\n\n", cmt.ID))
		if cmt.ParentID > 0 {
			b.WriteString(fmt.Sprintf("- **Parent:** Comment %d\n", cmt.ParentID))
		}
		if cmt.Author != "" {
			b.WriteString(fmt.Sprintf("- **Author:** %s\n", cmt.Author))
		}
		if cmt.Date != "" {
			b.WriteString(fmt.Sprintf("- **Date:** %s\n", cmt.Date))
		}
		if cmt.SectionID != "" {
			b.WriteString(fmt.Sprintf("- **Section:** `%s`\n", cmt.SectionID))
		}
		b.WriteString(fmt.Sprintf("- **Text:** %s\n\n", cmt.Text))
	}

	return b.String()
}

func RenderSectionFile(sec model.Section) string {
	return RenderSection(sec)
}
