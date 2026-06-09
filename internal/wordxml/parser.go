package wordxml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/SanD94/redline/internal/model"
)

const wordMLNamespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
const word10Namespace = "http://schemas.microsoft.com/office/word/2010/wordml"
const word12Namespace = "http://schemas.microsoft.com/office/word/2012/wordml"

type parser struct {
	decoder          *xml.Decoder
	changes          []model.Change
	openComments     map[int]bool
	headingStyles    map[string]int
	paraIdx          int
	changeParaIdx    map[int]int
	commentParaIdx   map[int]int
}

func Parse(documentXML, commentsXML, commentsExtendedXML, stylesXML []byte) (*model.RevealResult, error) {
	p := &parser{
		openComments:  make(map[int]bool),
		headingStyles: make(map[string]int),
		changeParaIdx:  make(map[int]int),
		commentParaIdx: make(map[int]int),
	}

	p.loadHeadingStyles(stylesXML)
	comments := p.parseComments(commentsXML, commentsExtendedXML)
	paras, err := p.parseBody(documentXML)
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}

	sections, sourceMap := buildSections(paras, p.headingStyles)

	p.assignLocations(paras, sections, comments)

	result := &model.RevealResult{
		Sections:  sections,
		Changes:   p.changes,
		Comments:  comments,
		SourceMap: sourceMap,
	}

	if result.Changes == nil {
		result.Changes = []model.Change{}
	}
	if result.Comments == nil {
		result.Comments = []model.Comment{}
	}

	return result, nil
}

func isWordElement(name xml.Name, local string) bool {
	return name.Space == wordMLNamespace && name.Local == local
}

func isInNS(name xml.Name, ns, local string) bool {
	return name.Space == ns && name.Local == local
}

func isWordEnd(name xml.Name, local string) bool {
	return name.Local == local
}

type paraData struct {
	style    string
	text     string
	changeID int
	ct       model.ChangeType
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
				p.skipToEnd("tbl")
			case isWordElement(t.Name, "sectPr"):
				p.skipToEnd("sectPr")
			default:
				p.skipToEnd(t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "body") {
				return paras, nil
			}
		}
	}
}

func (p *parser) readParagraph(start xml.StartElement) (paraData, error) {
	var pc paraData
	var textParts []string
	p.paraIdx++

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return pc, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case isWordElement(t.Name, "pPr"):
				pc.style = p.readStyle(t)
			case isWordElement(t.Name, "r"):
				p.readRun(t, false, false, &textParts)
			case isWordElement(t.Name, "ins"):
				p.readChange(t, true, &textParts)
			case isWordElement(t.Name, "del"):
				p.readChange(t, false, &textParts)
			case isWordElement(t.Name, "moveFrom"):
				p.readChange(t, false, &textParts)
			case isWordElement(t.Name, "moveTo"):
				p.readChange(t, true, &textParts)
			case isWordElement(t.Name, "moveFromRangeStart"):
				p.skipToEnd("moveFromRangeStart")
			case isWordElement(t.Name, "moveToRangeStart"):
				p.skipToEnd("moveToRangeStart")
			case isWordElement(t.Name, "moveFromRangeEnd"):
				p.skipToEnd("moveFromRangeEnd")
			case isWordElement(t.Name, "moveToRangeEnd"):
				p.skipToEnd("moveToRangeEnd")
			case isWordElement(t.Name, "commentRangeStart"):
				id := p.getIntAttr(t, "id")
				if id > 0 {
					p.openComments[id] = true
					p.commentParaIdx[id] = p.paraIdx
				}
				p.skipToEnd("commentRangeStart")
			case isWordElement(t.Name, "commentRangeEnd"):
				p.skipToEnd("commentRangeEnd")
			case isWordElement(t.Name, "commentReference"):
				p.skipToEnd("commentReference")
			case isWordElement(t.Name, "bookmarkStart"):
				p.skipToEnd("bookmarkStart")
			case isWordElement(t.Name, "bookmarkEnd"):
				p.skipToEnd("bookmarkEnd")
			default:
				p.skipToEnd(t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "p") {
				pc.text = strings.Join(textParts, "")
				return pc, nil
			}
		case xml.CharData:
		}
	}
}

func (p *parser) readStyle(start xml.StartElement) string {
	style := ""
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return style
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isWordElement(t.Name, "pStyle") {
				style = p.getAttr(t, "val")
			}
			p.skipToEnd(t.Name.Local)
		case xml.EndElement:
			if isWordEnd(t.Name, "pPr") {
				return style
			}
		}
	}
}

func (p *parser) readRun(start xml.StartElement, isInsert, isDelete bool, parts *[]string) {
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case isWordElement(t.Name, "t"):
				text := p.readCharData()
				if !isDelete {
					*parts = append(*parts, text)
				}
			case isWordElement(t.Name, "delText"):
				text := p.readCharData()
				if isDelete {
					*parts = append(*parts, text)
				}
			default:
				p.skipToEnd(t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "r") {
				return
			}
		case xml.CharData:
		}
	}
}

func (p *parser) readChange(start xml.StartElement, isInsert bool, parts *[]string) {
	var changeText []string

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isWordElement(t.Name, "r") {
				p.readRun(t, isInsert, !isInsert, &changeText)
			} else {
				p.skipToEnd(t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "ins") || isWordEnd(t.Name, "del") ||
				isWordEnd(t.Name, "moveFrom") || isWordEnd(t.Name, "moveTo") {
				text := strings.Join(changeText, "")
				if text != "" {
					if isInsert {
						*parts = append(*parts, "[++"+text+"++]")
					} else {
						*parts = append(*parts, "[--"+text+"--]")
					}
				}

				id := p.getIntAttr(start, "id")
				if id > 0 {
					ct := model.ChangeInsertion
					if !isInsert {
						ct = model.ChangeDeletion
					}
					p.changes = append(p.changes, model.Change{
						ID:     id,
						Type:   ct,
						Author: p.getAttr(start, "author"),
						Date:   p.getAttr(start, "date"),
						Text:   text,
					})
					p.changeParaIdx[id] = p.paraIdx
				}
				return
			}
		case xml.CharData:
		}
	}
}

func (p *parser) skipToEnd(localName string) {
	depth := 1
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			_ = t
		case xml.EndElement:
			depth--
			if depth <= 0 && isWordEnd(t.Name, localName) {
				return
			}
		}
	}
}

func (p *parser) readCharData() string {
	var text strings.Builder
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return text.String()
		}
		switch t := tok.(type) {
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			return text.String()
		default:
			return text.String()
		}
	}
}

func (p *parser) parseComments(commentsXML, commentsExtendedXML []byte) []model.Comment {
	if len(commentsXML) == 0 {
		return nil
	}

	comments := p.readCommentBodies(commentsXML)
	if comments == nil {
		return nil
	}

	threadMap := p.readCommentThreads(commentsExtendedXML)

	paraToCommentID := make(map[string]int)
	for _, cmt := range comments {
		paraToCommentID[cmt.paraID] = cmt.ID
	}

	for i, cmt := range comments {
		if parentParaID, ok := threadMap[cmt.paraID]; ok {
			if parentID, ok := paraToCommentID[parentParaID]; ok {
				comments[i].ParentID = parentID
			}
		}
	}

	result := make([]model.Comment, len(comments))
	for i, cmt := range comments {
		result[i] = model.Comment{
			ID:       cmt.ID,
			ParentID: cmt.ParentID,
			Author:   cmt.Author,
			Date:     cmt.Date,
			Text:     cmt.Text,
		}
	}
	return result
}

type commentData struct {
	ID       int
	ParentID int
	paraID   string
	Author   string
	Date     string
	Text     string
}

func (p *parser) readCommentBodies(commentsXML []byte) []commentData {
	decoder := xml.NewDecoder(strings.NewReader(string(commentsXML)))
	var comments []commentData

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isWordElement(t.Name, "comment") {
				cmt := commentData{
					ID:     p.getIntAttr(t, "id"),
					Author: p.getAttr(t, "author"),
					Date:   p.getAttr(t, "date"),
				}
				cmt.paraID, cmt.Text = p.readCommentContent(decoder)
				if cmt.ID > 0 {
					comments = append(comments, cmt)
				}
			}
		}
	}
	return comments
}

func (p *parser) readCommentContent(decoder *xml.Decoder) (paraID, text string) {
	var textParts []string
	var foundParaID string

	for {
		tok, err := decoder.Token()
		if err != nil {
			return foundParaID, strings.Join(textParts, "")
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isWordElement(t.Name, "t") {
				textParts = append(textParts, p.readCharDataFrom(decoder))
			} else if isWordElement(t.Name, "p") {
				if paraID := p.getAttrInNS(t, word10Namespace, "paraId"); paraID != "" && foundParaID == "" {
					foundParaID = paraID
				}
			} else if isWordElement(t.Name, "r") || isWordElement(t.Name, "p") {
				continue
			} else if isInNS(t.Name, word10Namespace, "paraId") {
				continue
			} else {
				p.skipToEndIn(decoder, t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "comment") {
				return foundParaID, strings.Join(textParts, "")
			}
		case xml.CharData:
		}
	}
}

func (p *parser) readCommentThreads(commentsExtendedXML []byte) map[string]string {
	threadMap := make(map[string]string)
	if len(commentsExtendedXML) == 0 {
		return threadMap
	}

	decoder := xml.NewDecoder(strings.NewReader(string(commentsExtendedXML)))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return threadMap
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isInNS(t.Name, word12Namespace, "commentEx") {
				paraID := p.getAttr(t, "paraId")
				parentParaID := p.getAttr(t, "paraIdParent")
				if paraID != "" && parentParaID != "" {
					threadMap[paraID] = parentParaID
				}
			}
		}
	}
	return threadMap
}

func (p *parser) readCharDataFrom(decoder *xml.Decoder) string {
	var text strings.Builder
	for {
		tok, err := decoder.Token()
		if err != nil {
			return text.String()
		}
		switch t := tok.(type) {
		case xml.CharData:
			text.Write(t)
		case xml.EndElement:
			return text.String()
		default:
			return text.String()
		}
	}
}

func (p *parser) skipToEndIn(decoder *xml.Decoder, localName string) {
	depth := 1
	for {
		tok, err := decoder.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			_ = t
		case xml.EndElement:
			depth--
			if depth <= 0 && isWordEnd(t.Name, localName) {
				return
			}
		}
	}
}

func (p *parser) loadHeadingStyles(stylesXML []byte) {
	if len(stylesXML) == 0 {
		return
	}

	decoder := xml.NewDecoder(strings.NewReader(string(stylesXML)))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if isWordElement(start.Name, "style") {
			styleID := p.getAttr(start, "styleId")
			if styleID == "" {
				continue
			}
			canonicalName := p.readStyleCanonicalName(decoder)
			if canonicalName != "" {
				if level := headingLevelFromName(canonicalName); level > 0 {
					p.headingStyles[styleID] = level
				}
			}
		}
	}
}

func (p *parser) readStyleCanonicalName(decoder *xml.Decoder) string {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return ""
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isWordElement(t.Name, "name") {
				return p.getAttr(t, "val")
			}
			p.skipToEndIn(decoder, t.Name.Local)
		case xml.EndElement:
			if isWordEnd(t.Name, "style") {
				return ""
			}
		}
	}
}

func (p *parser) getAttr(start xml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func (p *parser) getAttrInNS(start xml.StartElement, ns, local string) string {
	for _, attr := range start.Attr {
		if attr.Name.Space == ns && attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func (p *parser) getIntAttr(start xml.StartElement, name string) int {
	v := p.getAttr(start, name)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

func (p *parser) assignLocations(paras []paraData, sections []model.Section, comments []model.Comment) {
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

	for i := range p.changes {
		if paraIdx, ok := p.changeParaIdx[p.changes[i].ID]; ok {
			if paraIdx < len(paraToSection) {
				p.changes[i].SectionID = paraToSection[paraIdx]
			}
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

func buildSections(paras []paraData, headingStyles map[string]int) ([]model.Section, model.SourceMap) {
	var sections []model.Section
	sourceMap := make(model.SourceMap)
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
		sourceMap[id] = model.SourceLocation{
			DocxPath: fmt.Sprintf("word/document.xml#heading-%s", id),
		}
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

	return sections, sourceMap
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
