package model

type VersionMode int

const (
	VersionOld VersionMode = iota
	VersionNew
)

type Comment struct {
	ID         int    `json:"id"`
	ParentID   int    `json:"parentId,omitempty"`
	Author     string `json:"author,omitempty"`
	Date       string `json:"date,omitempty"`
	Text       string `json:"text"`
	SectionID  string `json:"sectionId,omitempty"`
	AnchorText string `json:"anchorText,omitempty"`
	AnchorKind string `json:"anchorKind,omitempty"`
}

type Section struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Level   int    `json:"level"`
	Content string `json:"-"`
}

type RevealResult struct {
	Sections []Section `json:"sections"`
	Comments []Comment `json:"comments"`
}
