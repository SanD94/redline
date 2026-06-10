package wordxml

import (
	"encoding/xml"
	"strconv"
	"strings"
)

const (
	wordMLNamespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	word10Namespace = "http://schemas.microsoft.com/office/word/2010/wordml"
	word12Namespace = "http://schemas.microsoft.com/office/word/2012/wordml"
)

func isWordElement(name xml.Name, local string) bool {
	return name.Space == wordMLNamespace && name.Local == local
}

func isInNS(name xml.Name, ns, local string) bool {
	return name.Space == ns && name.Local == local
}

func isWordEnd(name xml.Name, local string) bool {
	return name.Local == local
}

func skipToEnd(decoder *xml.Decoder, localName string) {
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

func readCharData(decoder *xml.Decoder) string {
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

func getAttr(start xml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func getAttrInNS(start xml.StartElement, ns, local string) string {
	for _, attr := range start.Attr {
		if attr.Name.Space == ns && attr.Name.Local == local {
			return attr.Value
		}
	}
	return ""
}

func getIntAttr(start xml.StartElement, name string) int {
	v := getAttr(start, name)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}
