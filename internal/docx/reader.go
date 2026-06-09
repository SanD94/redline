package docx

import (
	"archive/zip"
	"fmt"
	"io"
)

type Reader struct {
	documentXML         []byte
	commentsXML         []byte
	commentsExtendedXML []byte
	stylesXML           []byte
}

func Open(path string) (*Reader, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open docx archive: %w", err)
	}
	defer r.Close()

	rd := &Reader{}

	for _, f := range r.File {
		switch f.Name {
		case "word/document.xml":
			rd.documentXML, err = readZipFile(f)
			if err != nil {
				return nil, fmt.Errorf("read word/document.xml: %w", err)
			}
		case "word/comments.xml":
			rd.commentsXML, err = readZipFile(f)
			if err != nil {
				return nil, fmt.Errorf("read word/comments.xml: %w", err)
			}
		case "word/commentsExtended.xml":
			rd.commentsExtendedXML, err = readZipFile(f)
			if err != nil {
				return nil, fmt.Errorf("read word/commentsExtended.xml: %w", err)
			}
		case "word/styles.xml":
			rd.stylesXML, err = readZipFile(f)
			if err != nil {
				return nil, fmt.Errorf("read word/styles.xml: %w", err)
			}
		}
	}

	if rd.documentXML == nil {
		return nil, fmt.Errorf("word/document.xml not found in docx archive")
	}

	return rd, nil
}

func (r *Reader) DocumentXML() []byte {
	return r.documentXML
}

func (r *Reader) CommentsXML() []byte {
	return r.commentsXML
}

func (r *Reader) CommentsExtendedXML() []byte {
	return r.commentsExtendedXML
}

func (r *Reader) StylesXML() []byte {
	return r.stylesXML
}

func (r *Reader) HasComments() bool {
	return len(r.commentsXML) > 0
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
