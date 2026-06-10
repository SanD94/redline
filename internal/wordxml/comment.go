package wordxml

import (
	"encoding/xml"
	"io"
	"strings"

	"github.com/SanD94/redline/internal/model"
)

type commentData struct {
	ID       int
	ParentID int
	paraID   string
	Author   string
	Date     string
	Text     string
}

func parseComments(commentsXML, commentsExtendedXML []byte) []model.Comment {
	if len(commentsXML) == 0 {
		return nil
	}

	comments := readCommentBodies(commentsXML)
	if comments == nil {
		return nil
	}

	threadMap := readCommentThreads(commentsExtendedXML)

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

func readCommentBodies(commentsXML []byte) []commentData {
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
					ID:     getIntAttr(t, "id"),
					Author: getAttr(t, "author"),
					Date:   getAttr(t, "date"),
				}
				cmt.paraID, cmt.Text = readCommentContent(decoder)
				if cmt.ID > 0 {
					comments = append(comments, cmt)
				}
			}
		}
	}
	return comments
}

func readCommentContent(decoder *xml.Decoder) (paraID, text string) {
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
				textParts = append(textParts, readCharData(decoder))
			} else if isWordElement(t.Name, "p") {
				if paraID := getAttrInNS(t, word10Namespace, "paraId"); paraID != "" && foundParaID == "" {
					foundParaID = paraID
				}
			} else if isWordElement(t.Name, "r") || isWordElement(t.Name, "p") {
				continue
			} else if isInNS(t.Name, word10Namespace, "paraId") {
				continue
			} else {
				skipToEnd(decoder, t.Name.Local)
			}
		case xml.EndElement:
			if isWordEnd(t.Name, "comment") {
				return foundParaID, strings.Join(textParts, "")
			}
		case xml.CharData:
		}
	}
}

func readCommentThreads(commentsExtendedXML []byte) map[string]string {
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
				paraID := getAttr(t, "paraId")
				parentParaID := getAttr(t, "paraIdParent")
				if paraID != "" && parentParaID != "" {
					threadMap[paraID] = parentParaID
				}
			}
		}
	}
	return threadMap
}
