package model

import "encoding/json"

type VersionMode int

const (
	VersionOld VersionMode = iota
	VersionNew
)

type ChangeType string

const (
	ChangeInsertion ChangeType = "insertion"
	ChangeDeletion  ChangeType = "deletion"
)

type Change struct {
	ID        int        `json:"id"`
	Type      ChangeType `json:"type"`
	Author    string     `json:"author,omitempty"`
	Date      string     `json:"date,omitempty"`
	Text      string     `json:"text"`
	SectionID string     `json:"sectionId,omitempty"`
}

type Comment struct {
	ID        int    `json:"id"`
	ParentID  int    `json:"parentId,omitempty"`
	Author    string `json:"author,omitempty"`
	Date      string `json:"date,omitempty"`
	Text      string `json:"text"`
	SectionID string `json:"sectionId,omitempty"`
}

type Section struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Level   int    `json:"level"`
	Content string `json:"-"`
}

type SourceLocation struct {
	DocxPath    string `json:"docxPath"`
	ParagraphID int    `json:"paragraphId,omitempty"`
}

type SourceMap map[string]SourceLocation

type RevealResult struct {
	Sections  []Section    `json:"sections"`
	Changes   []Change     `json:"changes,omitempty"`
	Comments  []Comment    `json:"comments"`
	SourceMap SourceMap    `json:"sourceMap"`
}

func (r *RevealResult) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
