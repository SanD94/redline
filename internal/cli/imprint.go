package cli

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ImprintOptions struct {
	InputPath    string
	OutputPath   string
	WorkspaceDir string
}

func RunImprint(args []string) error {
	opts := ImprintOptions{WorkspaceDir: "."}
	if len(args) < 1 {
		return fmt.Errorf("usage: redline imprint <file.docx> --output <patched.docx> [--workspace <dir>]")
	}
	opts.InputPath = args[0]
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--output", "-o":
			if i+1 >= len(args) {
				return fmt.Errorf("usage: redline imprint <file.docx> --output <patched.docx> [--workspace <dir>]")
			}
			i++
			opts.OutputPath = args[i]
		case "--workspace", "-w":
			if i+1 >= len(args) {
				return fmt.Errorf("usage: redline imprint <file.docx> --output <patched.docx> [--workspace <dir>]")
			}
			i++
			opts.WorkspaceDir = args[i]
		default:
			return fmt.Errorf("unknown imprint option: %s", args[i])
		}
	}
	if opts.OutputPath == "" {
		return fmt.Errorf("imprint requires --output <patched.docx>")
	}
	if samePath(opts.InputPath, opts.OutputPath) {
		return fmt.Errorf("imprint output must not overwrite input docx")
	}

	markdown, err := readWorkspaceDocument(opts.WorkspaceDir)
	if err != nil {
		return err
	}
	warnings := []map[string]string{{
		"type":    "formatting-loss",
		"message": "imprint replaces document body text from Markdown and preserves non-document DOCX parts; complex Word formatting in replaced sections is not preserved yet",
	}}
	if err := patchDocxDocument(opts.InputPath, opts.OutputPath, markdownToDocumentXML(markdown)); err != nil {
		return err
	}
	if err := writeImprintWarnings(opts.OutputPath, warnings); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "wrote patched docx: %s\n", opts.OutputPath)
	fmt.Fprintf(os.Stderr, "warning: %s\n", warnings[0]["message"])
	return nil
}

func readWorkspaceDocument(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "document.md"))
	if err == nil {
		return string(data), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read document.md: %w", err)
	}
	return "", fmt.Errorf("workspace document.md not found; run redline reveal first")
}

func patchDocxDocument(inputPath, outputPath string, documentXML []byte) error {
	r, err := zip.OpenReader(inputPath)
	if err != nil {
		return fmt.Errorf("open source docx: %w", err)
	}
	defer r.Close()

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(outputPath), ".redline-imprint-*.docx")
	if err != nil {
		return fmt.Errorf("create temp docx: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	zw := zip.NewWriter(tmp)
	replaced := false
	for _, f := range r.File {
		h := f.FileHeader
		w, err := zw.CreateHeader(&h)
		if err != nil {
			zw.Close()
			tmp.Close()
			return fmt.Errorf("create zip entry %s: %w", f.Name, err)
		}
		if f.Name == "word/document.xml" {
			if _, err := w.Write(documentXML); err != nil {
				zw.Close()
				tmp.Close()
				return fmt.Errorf("write document.xml: %w", err)
			}
			replaced = true
			continue
		}
		rc, err := f.Open()
		if err != nil {
			zw.Close()
			tmp.Close()
			return fmt.Errorf("open zip entry %s: %w", f.Name, err)
		}
		_, copyErr := io.Copy(w, rc)
		closeErr := rc.Close()
		if copyErr != nil {
			zw.Close()
			tmp.Close()
			return fmt.Errorf("copy zip entry %s: %w", f.Name, copyErr)
		}
		if closeErr != nil {
			zw.Close()
			tmp.Close()
			return fmt.Errorf("close zip entry %s: %w", f.Name, closeErr)
		}
	}
	if err := zw.Close(); err != nil {
		tmp.Close()
		return fmt.Errorf("close output docx: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp docx: %w", err)
	}
	if !replaced {
		return fmt.Errorf("source docx missing word/document.xml")
	}
	if err := validateDocx(tmpPath); err != nil {
		return err
	}
	return os.Rename(tmpPath, outputPath)
}

func markdownToDocumentXML(markdown string) []byte {
	var body strings.Builder
	for _, block := range markdownBlocks(markdown) {
		level, text := markdownHeading(block)
		if level > 0 {
			body.WriteString(`<w:p><w:pPr><w:pStyle w:val="Heading`)
			body.WriteString(fmt.Sprint(level))
			body.WriteString(`"/></w:pPr><w:r><w:t>`)
			body.WriteString(xmlEscape(text))
			body.WriteString(`</w:t></w:r></w:p>`)
			continue
		}
		body.WriteString(`<w:p><w:r><w:t>`)
		body.WriteString(xmlEscape(strings.ReplaceAll(block, "\n", " ")))
		body.WriteString(`</w:t></w:r></w:p>`)
	}
	return []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` +
		`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>` +
		body.String() + `<w:sectPr/></w:body></w:document>`)
}

func markdownBlocks(markdown string) []string {
	var blocks []string
	for _, block := range strings.Split(strings.ReplaceAll(markdown, "\r\n", "\n"), "\n\n") {
		block = strings.TrimSpace(block)
		if block != "" {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func markdownHeading(block string) (int, string) {
	line := strings.TrimSpace(strings.Split(block, "\n")[0])
	level := 0
	for level < len(line) && line[level] == '#' {
		level++
	}
	if level == 0 || level > 9 || level >= len(line) || line[level] != ' ' {
		return 0, ""
	}
	return level, strings.TrimSpace(line[level:])
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(s)); err != nil {
		return s
	}
	return b.String()
}

func validateDocx(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("validate output docx: %w", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			return nil
		}
	}
	return fmt.Errorf("validate output docx: missing word/document.xml")
}

func writeImprintWarnings(outputPath string, warnings []map[string]string) error {
	data, err := json.MarshalIndent(map[string]any{"warnings": warnings}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal imprint warnings: %w", err)
	}
	path := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".warnings.json"
	return os.WriteFile(path, data, 0644)
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA == nil && errB == nil {
		return absA == absB
	}
	return a == b
}
