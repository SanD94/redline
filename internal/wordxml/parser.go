package wordxml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/SanD94/redline/internal/model"
)

type parser struct {
	decoder        *xml.Decoder
	openComments   map[int]bool
	headingStyles  map[string]int
	paraIdx        int
	commentParaIdx map[int]int
	version        model.VersionMode
}

type paraData struct {
	style string
	text  string
}

func Parse(documentXML, commentsXML, commentsExtendedXML, stylesXML []byte, version model.VersionMode) (*model.RevealResult, error) {
	p := &parser{
		openComments:   make(map[int]bool),
		headingStyles:  make(map[string]int),
		commentParaIdx: make(map[int]int),
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

	result := &model.RevealResult{
		Sections: sections,
		Comments: comments,
	}

	if result.Comments == nil {
		result.Comments = []model.Comment{}
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

	for i := range comments {
		if paraIdx, ok := p.commentParaIdx[comments[i].ID]; ok {
			if paraIdx < len(paraToSection) {
				comments[i].SectionID = paraToSection[paraIdx]
			}
		}
	}
}
