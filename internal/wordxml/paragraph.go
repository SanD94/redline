package wordxml

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/SanD94/redline/internal/model"
)

func (p *parser) readParagraph(start xml.StartElement) (paraData, error) {
	var pc paraData
	var textParts []string
	p.paraIdx++
	p.textRunInPara = 0
	pc.idx = p.paraIdx

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
				p.readChangeContent(t, "added", p.version == model.VersionNew, false, &textParts)
			case isWordElement(t.Name, "del"):
				p.readChangeContent(t, "deleted", p.version == model.VersionOld, true, &textParts)
			case isWordElement(t.Name, "moveFrom"):
				text := p.readChangeContent(t, "deleted", p.version == model.VersionOld, false, &textParts)
				p.appendMoveText("from", t, text)
			case isWordElement(t.Name, "moveTo"):
				text := p.readChangeContent(t, "added", p.version == model.VersionNew, false, &textParts)
				p.appendMoveText("to", t, text)
			case isWordElement(t.Name, "moveFromRangeStart"):
				p.startMoveRange("from", t)
				skipToEnd(p.decoder, "moveFromRangeStart")
			case isWordElement(t.Name, "moveToRangeStart"):
				p.startMoveRange("to", t)
				skipToEnd(p.decoder, "moveToRangeStart")
			case isWordElement(t.Name, "moveFromRangeEnd"):
				p.activeMoveFrom = ""
				skipToEnd(p.decoder, "moveFromRangeEnd")
			case isWordElement(t.Name, "moveToRangeEnd"):
				p.activeMoveTo = ""
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
				pc.text = normalizeParagraphText(strings.Join(textParts, ""))
				return pc, nil
			}
		case xml.CharData:
		}
	}
}

func normalizeParagraphText(text string) string {
	text = strings.ReplaceAll(text, "\u00a0", " ")
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	text = strings.Join(fields, " ")
	for _, punct := range []string{".", ",", ";", ":", "!", "?"} {
		text = strings.ReplaceAll(text, " "+punct, punct)
	}
	return text
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

func (p *parser) readRun(start xml.StartElement, anchorKind string, emit, useDelText bool, parts *[]string) string {
	var runText strings.Builder

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return runText.String()
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case isWordElement(t.Name, "t"):
				text := readCharData(p.decoder)
				p.appendTextRun(anchorKind, text)
				p.appendCommentAnchorText(anchorKind, text)
				if !useDelText {
					runText.WriteString(text)
				}
				if emit && !useDelText {
					*parts = append(*parts, text)
				}
			case isWordElement(t.Name, "delText"):
				text := readCharData(p.decoder)
				p.appendTextRun(anchorKind, text)
				p.appendCommentAnchorText(anchorKind, text)
				if useDelText {
					runText.WriteString(text)
				}
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
				return runText.String()
			}
		case xml.CharData:
		}
	}
}

func (p *parser) readChangeContent(start xml.StartElement, anchorKind string, emit, useDelText bool, parts *[]string) string {
	var changeText strings.Builder

	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return changeText.String()
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch {
			case isWordElement(t.Name, "r"):
				changeText.WriteString(p.readRun(t, anchorKind, emit, useDelText, parts))
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
				return changeText.String()
			}
		}
	}
	return changeText.String()
}

func (p *parser) startMoveRange(side string, start xml.StartElement) {
	key := moveKey(start)
	if key == "" {
		return
	}
	mv := p.ensureMove(key, start)
	if side == "from" {
		p.activeMoveFrom = key
		if mv.FromParaIdx == 0 {
			mv.FromParaIdx = p.paraIdx
		}
	} else {
		p.activeMoveTo = key
		if mv.ToParaIdx == 0 {
			mv.ToParaIdx = p.paraIdx
		}
	}
}

func (p *parser) appendMoveText(side string, start xml.StartElement, text string) {
	if text == "" {
		return
	}
	key := ""
	if side == "from" {
		key = p.activeMoveFrom
	} else {
		key = p.activeMoveTo
	}
	if key == "" {
		key = moveKey(start)
	}
	if key == "" {
		return
	}
	mv := p.ensureMove(key, start)
	if side == "from" {
		if mv.FromParaIdx == 0 {
			mv.FromParaIdx = p.paraIdx
		}
		mv.FromText.WriteString(text)
	} else {
		if mv.ToParaIdx == 0 {
			mv.ToParaIdx = p.paraIdx
		}
		mv.ToText.WriteString(text)
	}
}

func (p *parser) ensureMove(key string, start xml.StartElement) *moveData {
	mv := p.moves[key]
	if mv == nil {
		mv = &moveData{ID: getAttr(start, "id"), Name: getAttr(start, "name")}
		p.moves[key] = mv
		p.moveOrder = append(p.moveOrder, key)
	}
	if mv.ID == "" {
		mv.ID = getAttr(start, "id")
	}
	if mv.Name == "" {
		mv.Name = getAttr(start, "name")
	}
	if mv.Author == "" {
		mv.Author = getAttr(start, "author")
	}
	if mv.Date == "" {
		mv.Date = getAttr(start, "date")
	}
	return mv
}

func moveKey(start xml.StartElement) string {
	if name := getAttr(start, "name"); name != "" {
		return "name:" + name
	}
	if id := getAttr(start, "id"); id != "" {
		return "id:" + id
	}
	return ""
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

func (p *parser) appendTextRun(kind, text string) {
	if text == "" {
		return
	}
	p.textRunIdx++
	p.textRunInPara++
	p.textRuns = append(p.textRuns, model.TextRun{
		ID:            fmt.Sprintf("run-%06d", p.textRunIdx),
		BlockID:       stableBlockID(p.paraIdx),
		Kind:          kind,
		Text:          text,
		SourcePointer: textRunSourcePointer(p.paraIdx, p.textRunInPara),
	})
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
