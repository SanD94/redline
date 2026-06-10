package wordxml

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/SanD94/redline/internal/model"
)

func buildSections(paras []paraData, headingStyles map[string]int) []model.Section {
	var sections []model.Section
	sectionID := 0

	var currentSection *model.Section

	startSection := func(title string) {
		id := slugify(title)
		if id == "" {
			id = fmt.Sprintf("section-%d", sectionID)
		}
		sec := model.Section{
			ID:    id,
			Title: title,
			Level: 1,
		}
		sectionID++
		currentSection = &sec
	}

	startSection("")

	for _, para := range paras {
		level := headingLevel(para.style, headingStyles)
		if level == 1 {
			if currentSection != nil && strings.TrimSpace(currentSection.Content) != "" {
				sections = append(sections, *currentSection)
			}
			startSection(para.text)
		} else if level >= 2 {
			prefix := strings.Repeat("#", level)
			if currentSection.Content != "" {
				currentSection.Content += "\n\n"
			}
			currentSection.Content += prefix + " " + para.text
		} else if strings.TrimSpace(para.text) != "" {
			if currentSection.Content != "" {
				currentSection.Content += "\n\n"
			}
			currentSection.Content += para.text
		}
	}

	if currentSection != nil {
		if strings.TrimSpace(currentSection.Content) != "" || len(sections) == 0 {
			sections = append(sections, *currentSection)
		}
	}

	if len(sections) == 0 {
		sections = append(sections, model.Section{
			ID:    "document",
			Title: "",
			Level: 1,
		})
	}

	return sections
}

func headingLevel(styleID string, headingStyles map[string]int) int {
	if level, ok := headingStyles[styleID]; ok {
		return level
	}
	return headingLevelFromName(styleID)
}

func headingLevelFromName(name string) int {
	s := name
	if strings.HasPrefix(s, "heading ") {
		s = strings.TrimPrefix(s, "heading ")
	} else if strings.HasPrefix(s, "Heading ") {
		s = strings.TrimPrefix(s, "Heading ")
	} else {
		return 0
	}
	level, err := strconv.Atoi(s)
	if err != nil || level < 1 || level > 9 {
		return 1
	}
	return level
}

func slugify(s string) string {
	var result strings.Builder
	s = strings.ToLower(s)
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == ' ' {
			if r == ' ' {
				result.WriteRune('-')
			} else {
				result.WriteRune(r)
			}
		}
	}
	slug := strings.Trim(result.String(), "-")
	if slug == "" {
		slug = "untitled"
	}
	return slug
}
