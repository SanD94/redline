package wordxml

import (
	"encoding/xml"
	"io"
	"strings"
)

func loadHeadingStyles(stylesXML []byte, headingStyles map[string]int) {
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
			styleID := getAttr(start, "styleId")
			if styleID == "" {
				continue
			}
			canonicalName := readStyleCanonicalName(decoder)
			if canonicalName != "" {
				if level := headingLevelFromName(canonicalName); level > 0 {
					headingStyles[styleID] = level
				}
			}
		}
	}
}

func readStyleCanonicalName(decoder *xml.Decoder) string {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return ""
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if isWordElement(t.Name, "name") {
				return getAttr(t, "val")
			}
			skipToEnd(decoder, t.Name.Local)
		case xml.EndElement:
			if isWordEnd(t.Name, "style") {
				return ""
			}
		}
	}
}
