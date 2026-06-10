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
				p.readRun(t, false, false, &textParts)
			case isWordElement(t.Name, "ins"):
				if p.version == model.VersionNew {
					p.readChangeContent(true, &textParts)
				} else {
					skipToEnd(p.decoder, "ins")
				}
			case isWordElement(t.Name, "del"):
				if p.version == model.VersionOld {
					p.readChangeContent(false, &textParts)
				} else {
					skipToEnd(p.decoder, "del")
				}
			case isWordElement(t.Name, "moveFrom"):
				if p.version == model.VersionOld {
					p.readChangeContent(false, &textParts)
				} else {
					skipToEnd(p.decoder, "moveFrom")
				}
			case isWordElement(t.Name, "moveTo"):
				if p.version == model.VersionNew {
					p.readChangeContent(true, &textParts)
				} else {
					skipToEnd(p.decoder, "moveTo")
				}
			case isWordElement(t.Name, "moveFromRangeStart"):
				skipToEnd(p.decoder, "moveFromRangeStart")
			case isWordElement(t.Name, "moveToRangeStart"):
				skipToEnd(p.decoder, "moveToRangeStart")
			case isWordElement(t.Name, "moveFromRangeEnd"):
				skipToEnd(p.decoder, "moveFromRangeEnd")
			case isWordElement(t.Name, "moveToRangeEnd"):
				skipToEnd(p.decoder, "moveToRangeEnd")
			case isWordElement(t.Name, "commentRangeStart"):
				id := getIntAttr(t, "id")
				if id > 0 {
					p.openComments[id] = true
					p.commentParaIdx[id] = p.paraIdx
				}
				skipToEnd(p.decoder, "commentRangeStart")
			case isWordElement(t.Name, "commentRangeEnd"):
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
				text := readCharData(p.decoder)
				if !isDelete {
					*parts = append(*parts, text)
				}
			case isWordElement(t.Name, "delText"):
				text := readCharData(p.decoder)
				if isDelete {
					*parts = append(*parts, text)
				}
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

func (p *parser) readChangeContent(isInsert bool, parts *[]string) {
	for {
		tok, err := p.decoder.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isWordElement(t.Name, "r") {
				p.readRun(t, isInsert, !isInsert, parts)
			} else {
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
