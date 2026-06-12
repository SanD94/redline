package wordxml

import (
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/SanD94/redline/internal/model"
)

type parser struct {
	decoder        *xml.Decoder
	openComments   map[int]bool
	commentAnchors map[int]*commentAnchor
	headingStyles  map[string]int
	paraIdx        int
	commentParaIdx map[int]int
	moves          map[string]*moveData
	moveOrder      []string
	activeMoveFrom string
	activeMoveTo   string
	version        model.VersionMode
}

type paraData struct {
	style string
	text  string
}

type commentAnchor struct {
	ParaIdx int
	Kind    string
	Text    strings.Builder
}

type moveData struct {
	ID          string
	Name        string
	Author      string
	Date        string
	FromParaIdx int
	ToParaIdx   int
	FromText    strings.Builder
	ToText      strings.Builder
}

func Parse(documentXML, commentsXML, commentsExtendedXML, stylesXML []byte, version model.VersionMode) (*model.RevealResult, error) {
	p := &parser{
		openComments:   make(map[int]bool),
		commentAnchors: make(map[int]*commentAnchor),
		headingStyles:  make(map[string]int),
		commentParaIdx: make(map[int]int),
		moves:          make(map[string]*moveData),
		version:        version,
	}

	loadHeadingStyles(stylesXML, p.headingStyles)
	comments := parseComments(commentsXML, commentsExtendedXML)
	paras, err := p.parseBody(documentXML)
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}

	sections := buildSections(paras, p.headingStyles)
	p.assignCommentLocations(paras, sections, comments)
	moves := p.buildMoves(paras, sections)

	result := &model.RevealResult{
		Sections: sections,
		Comments: comments,
		Moves:    moves,
	}

	if result.Comments == nil {
		result.Comments = []model.Comment{}
	}
	if result.Moves == nil {
		result.Moves = []model.Move{}
	}

	return result, nil
}

func (p *parser) parseBody(documentXML []byte) ([]paraData, error) {
	p.decoder = xml.NewDecoder(strings.NewReader(string(documentXML)))

	for {
		tok, err := p.decoder.Token()
		if err == io.EOF {
			return nil, fmt.Errorf("unexpected end of document: no body element")
		}
		if err != nil {
			return nil, fmt.Errorf("decode: %w", err)
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if isWordElement(start.Name, "body") {
			return p.readBody()
		}
	}
}

func (p *parser) readBody() ([]paraData, error) {
	var paras []paraData

	for {
		tok, err := p.decoder.Token()
		if err == io.EOF {
			return paras, nil
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case isWordElement(t.Name, "p"):
				para, err := p.readParagraph(t)
				if err != nil {
					return nil, err
				}
				paras = append(paras, para)
			case isWordElement(t.Name, "tbl"):
				skipToEnd(p.decoder, "tbl")
			case isWordElement(t.Name, "sectPr"):
				skipToEnd(p.decoder, "sectPr")
			default:
				skipToEnd(p.decoder, t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "body") {
				return paras, nil
			}
		}
	}
}

func (p *parser) assignCommentLocations(paras []paraData, sections []model.Section, comments []model.Comment) {
	paraToSection := p.paraToSection(paras, sections)

	for i := range comments {
		if paraIdx, ok := p.commentParaIdx[comments[i].ID]; ok {
			if paraIdx < len(paraToSection) {
				comments[i].SectionID = paraToSection[paraIdx]
			}
		}
		if anchor, ok := p.commentAnchors[comments[i].ID]; ok {
			comments[i].AnchorText = anchor.Text.String()
			comments[i].AnchorKind = anchor.Kind
		}
	}
}

func (p *parser) paraToSection(paras []paraData, sections []model.Section) []string {
	paraToSection := make([]string, len(paras)+2)
	secIdx := 0

	for i, para := range paras {
		level := headingLevel(para.style, p.headingStyles)
		if level == 1 && secIdx < len(sections)-1 {
			secIdx++
		}
		if secIdx < len(sections) {
			paraToSection[i+1] = sections[secIdx].ID
		}
	}
	return paraToSection
}

func (p *parser) buildMoves(paras []paraData, sections []model.Section) []model.Move {
	if len(p.moves) == 0 {
		return nil
	}

	paraToSection := p.paraToSection(paras, sections)
	keys := append([]string(nil), p.moveOrder...)
	seen := make(map[string]bool, len(keys))
	for _, key := range keys {
		seen[key] = true
	}
	for key := range p.moves {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	moves := make([]model.Move, 0, len(keys))
	for _, key := range keys {
		mv := p.moves[key]
		if mv == nil || mv.FromText.Len() == 0 || mv.ToText.Len() == 0 {
			continue
		}
		out := model.Move{
			ID:     mv.ID,
			Name:   mv.Name,
			Author: mv.Author,
			Date:   mv.Date,
			Text:   mv.FromText.String(),
			Source: "word/document.xml w:moveFrom/w:moveTo",
		}
		if out.ID == "" {
			out.ID = key
		}
		if mv.FromParaIdx < len(paraToSection) {
			out.FromSectionID = paraToSection[mv.FromParaIdx]
		}
		if mv.ToParaIdx < len(paraToSection) {
			out.ToSectionID = paraToSection[mv.ToParaIdx]
		}
		if mv.FromParaIdx > 0 && mv.FromParaIdx <= len(paras) {
			out.FromContext = paras[mv.FromParaIdx-1].text
		}
		if mv.ToParaIdx > 0 && mv.ToParaIdx <= len(paras) {
			out.ToContext = paras[mv.ToParaIdx-1].text
		}
		moves = append(moves, out)
	}
	return moves
}
