package model

type VersionMode int

const (
	VersionOld VersionMode = iota
	VersionNew
)

type Comment struct {
	ID            int    `json:"id"`
	StableID      string `json:"stableId,omitempty"`
	ParentID      int    `json:"parentId,omitempty"`
	Author        string `json:"author,omitempty"`
	Date          string `json:"date,omitempty"`
	Text          string `json:"text"`
	SectionID     string `json:"sectionId,omitempty"`
	AnchorText    string `json:"anchorText,omitempty"`
	AnchorKind    string `json:"anchorKind,omitempty"`
	AnchorRangeID string `json:"anchorRangeId,omitempty"`
	SourcePointer string `json:"sourcePointer,omitempty"`
	Context       string `json:"context,omitempty"`
}

type DocumentBlock struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	SectionID     string `json:"sectionId,omitempty"`
	Text          string `json:"text"`
	SourcePointer string `json:"sourcePointer"`
	Context       string `json:"context,omitempty"`
}

type TextRun struct {
	ID            string `json:"id"`
	BlockID       string `json:"blockId"`
	Kind          string `json:"kind"`
	Text          string `json:"text"`
	SourcePointer string `json:"sourcePointer"`
	Context       string `json:"context,omitempty"`
}

type AnchorRange struct {
	ID            string `json:"id"`
	CommentID     int    `json:"commentId"`
	SectionID     string `json:"sectionId,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Text          string `json:"text,omitempty"`
	SourcePointer string `json:"sourcePointer"`
	Context       string `json:"context,omitempty"`
}

type Warning struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Message       string `json:"message"`
	SourcePointer string `json:"sourcePointer,omitempty"`
	Context       string `json:"context,omitempty"`
}

type Move struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	Author        string `json:"author,omitempty"`
	Date          string `json:"date,omitempty"`
	Text          string `json:"text"`
	FromSectionID string `json:"fromSectionId,omitempty"`
	ToSectionID   string `json:"toSectionId,omitempty"`
	FromContext   string `json:"fromContext,omitempty"`
	ToContext     string `json:"toContext,omitempty"`
	Source        string `json:"source"`
}

type Section struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Level   int    `json:"level"`
	Content string `json:"-"`
}

type SourceModel struct {
	DocumentBlocks []DocumentBlock `json:"documentBlocks"`
	TextRuns       []TextRun       `json:"textRuns"`
	Comments       []Comment       `json:"comments"`
	AnchorRanges   []AnchorRange   `json:"anchorRanges"`
	ReviewActions  []Move          `json:"reviewActions"`
	Warnings       []Warning       `json:"warnings"`
}

type RevealResult struct {
	Sections     []Section       `json:"sections"`
	Comments     []Comment       `json:"comments"`
	Moves        []Move          `json:"moves"`
	Blocks       []DocumentBlock `json:"blocks"`
	TextRuns     []TextRun       `json:"textRuns"`
	AnchorRanges []AnchorRange   `json:"anchorRanges"`
	Warnings     []Warning       `json:"warnings"`
}

func (r *RevealResult) SourceModel() SourceModel {
	return SourceModel{
		DocumentBlocks: r.Blocks,
		TextRuns:       r.TextRuns,
		Comments:       r.Comments,
		AnchorRanges:   r.AnchorRanges,
		ReviewActions:  r.Moves,
		Warnings:       r.Warnings,
	}
}
