package wordxml

import (
	"encoding/xml"
	"strings"

	"github.com/SanD94/redline/internal/model"
)

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
				p.readRun(t, "normal", true, false, &textParts)
			case isWordElement(t.Name, "ins"):
				p.readChangeContent("added", p.version == model.VersionNew, false, &textParts)
			case isWordElement(t.Name, "del"):
				p.readChangeContent("deleted", p.version == model.VersionOld, true, &textParts)
			case isWordElement(t.Name, "moveFrom"):
				p.readChangeContent("deleted", p.version == model.VersionOld, false, &textParts)
			case isWordElement(t.Name, "moveTo"):
				p.readChangeContent("added", p.version == model.VersionNew, false, &textParts)
			case isWordElement(t.Name, "moveFromRangeStart"):
				skipToEnd(p.decoder, "moveFromRangeStart")
			case isWordElement(t.Name, "moveToRangeStart"):
				skipToEnd(p.decoder, "moveToRangeStart")
			case isWordElement(t.Name, "moveFromRangeEnd"):
				skipToEnd(p.decoder, "moveFromRangeEnd")
			case isWordElement(t.Name, "moveToRangeEnd"):
				skipToEnd(p.decoder, "moveToRangeEnd")
			case isWordElement(t.Name, "commentRangeStart"):
				p.startCommentAnchor(t, "normal")
				skipToEnd(p.decoder, "commentRangeStart")
			case isWordElement(t.Name, "commentRangeEnd"):
				p.endCommentAnchor(t)
				skipToEnd(p.decoder, "commentRangeEnd")
			case isWordElement(t.Name, "commentReference"):
				skipToEnd(p.decoder, "commentReference")
			case isWordElement(t.Name, "bookmarkStart"):
				skipToEnd(p.decoder, "bookmarkStart")
			case isWordElement(t.Name, "bookmarkEnd"):
				skipToEnd(p.decoder, "bookmarkEnd")
			default:
				skipToEnd(p.decoder, t.Name.Local)
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
				style = getAttr(t, "val")
			}
			skipToEnd(p.decoder, t.Name.Local)
		case xml.EndElement:
			if isWordEnd(t.Name, "pPr") {
				return style
			}
		}
	}
}

func (p *parser) readRun(start xml.StartElement, anchorKind string, emit, useDelText bool, parts *[]string) {
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case isWordElement(t.Name, "t"):
				text := readCharData(p.decoder)
				p.appendCommentAnchorText(anchorKind, text)
				if emit && !useDelText {
					*parts = append(*parts, text)
				}
			case isWordElement(t.Name, "delText"):
				text := readCharData(p.decoder)
				p.appendCommentAnchorText(anchorKind, text)
				if emit && useDelText {
					*parts = append(*parts, text)
				}
			case isWordElement(t.Name, "commentRangeStart"):
				p.startCommentAnchor(t, anchorKind)
				skipToEnd(p.decoder, "commentRangeStart")
			case isWordElement(t.Name, "commentRangeEnd"):
				p.endCommentAnchor(t)
				skipToEnd(p.decoder, "commentRangeEnd")
			case isWordElement(t.Name, "commentReference"):
				skipToEnd(p.decoder, "commentReference")
			default:
				skipToEnd(p.decoder, t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "r") {
				return
			}
		case xml.CharData:
		}
	}
}

func (p *parser) readChangeContent(anchorKind string, emit, useDelText bool, parts *[]string) {
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case isWordElement(t.Name, "r"):
				p.readRun(t, anchorKind, emit, useDelText, parts)
			case isWordElement(t.Name, "commentRangeStart"):
				p.startCommentAnchor(t, anchorKind)
				skipToEnd(p.decoder, "commentRangeStart")
			case isWordElement(t.Name, "commentRangeEnd"):
				p.endCommentAnchor(t)
				skipToEnd(p.decoder, "commentRangeEnd")
			case isWordElement(t.Name, "commentReference"):
				skipToEnd(p.decoder, "commentReference")
			default:
				skipToEnd(p.decoder, t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "ins") || isWordEnd(t.Name, "del") ||
				isWordEnd(t.Name, "moveFrom") || isWordEnd(t.Name, "moveTo") {
				return
			}
		}
	}
}

func (p *parser) startCommentAnchor(start xml.StartElement, kind string) {
	id := getIntAttr(start, "id")
	if id <= 0 {
		return
	}
	p.openComments[id] = true
	if _, ok := p.commentParaIdx[id]; !ok {
		p.commentParaIdx[id] = p.paraIdx
	}
	anchor := p.commentAnchors[id]
	if anchor == nil {
		anchor = &commentAnchor{ParaIdx: p.paraIdx}
		p.commentAnchors[id] = anchor
	} else if anchor.Text.Len() > 0 {
		anchor.Kind = mergeAnchorKind(anchor.Kind, kind)
	}
}

func (p *parser) endCommentAnchor(end xml.StartElement) {
	id := getIntAttr(end, "id")
	if id <= 0 {
		return
	}
	delete(p.openComments, id)
}

func (p *parser) appendCommentAnchorText(kind, text string) {
	if text == "" || len(p.openComments) == 0 {
		return
	}
	for id := range p.openComments {
		anchor := p.commentAnchors[id]
		if anchor == nil {
			anchor = &commentAnchor{ParaIdx: p.paraIdx, Kind: kind}
			p.commentAnchors[id] = anchor
		}
		anchor.Kind = mergeAnchorKind(anchor.Kind, kind)
		anchor.Text.WriteString(text)
	}
}

func mergeAnchorKind(existing, next string) string {
	if existing == "" {
		return next
	}
	if next == "" || existing == next {
		return existing
	}
	return "mixed"
}
