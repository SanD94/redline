package wordxml

import "fmt"

func stableBlockID(paraIdx int) string {
	return fmt.Sprintf("block-%04d", paraIdx)
}

func stableCommentID(commentID int) string {
	return fmt.Sprintf("comment-%d", commentID)
}

func stableAnchorID(commentID int) string {
	return fmt.Sprintf("anchor-%d", commentID)
}

func paragraphSourcePointer(paraIdx int) string {
	return fmt.Sprintf("word/document.xml#/w:document/w:body/w:p[%d]", paraIdx)
}

func textRunSourcePointer(paraIdx, runIdx int) string {
	return fmt.Sprintf("word/document.xml#/w:document/w:body/w:p[%d]/textRun[%d]", paraIdx, runIdx)
}

func commentSourcePointer(commentID int) string {
	return fmt.Sprintf("word/comments.xml#/w:comments/w:comment[@w:id='%d']", commentID)
}

func commentAnchorSourcePointer(paraIdx, commentID int) string {
	return fmt.Sprintf("word/document.xml#/w:document/w:body/w:p[%d]/w:commentRangeStart[@w:id='%d']", paraIdx, commentID)
}

func blockIndexFromID(id string) int {
	var idx int
	if _, err := fmt.Sscanf(id, "block-%04d", &idx); err != nil {
		return 0
	}
	return idx
}
