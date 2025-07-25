package reports

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"strings"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/parser"
	"github.com/raykov/mdtopdf"
)

//go:embed templates/*.tmpl
var tmplFS embed.FS

var sharedFuncMap = template.FuncMap{
	"add":      add,
	"pad":      pad,
	"padNum":   padNum,
	"padFloat": padFloat,
	"fmtDur":   parser.FmtDuration,
}

func add(a, b int) int { return a + b }

func markdownToPdf(markdown string, pdfFile string) error {
	md := strings.NewReader(markdown)

	out, err := os.Create(pdfFile)
	if err != nil {
		return fmt.Errorf("creating PDF file: %w", err)
	}
	defer out.Close()

	if err := mdtopdf.Convert(md, out); err != nil {
		return fmt.Errorf("conversion failed: %v", err)
	}

	return nil
}

// pad right-spaces a string to width w
func pad(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	return fmt.Sprintf("%-*s", w, s)
	// return s + string(bytes.Repeat([]byte(" "), w-len(s)))
}

func padFloat(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return fmt.Sprintf("%*s", w, s)
	// return string(bytes.Repeat([]byte(" "), w-len(s))) + s
}

func padNum(n int64, w int) string {
	s := fmt.Sprint(n)
	if len(s) >= w {
		return s
	}
	return fmt.Sprintf("%*s", w, s)
	// return string(bytes.Repeat([]byte(" "), w-len(s))) + s
}

func writeMarkdown(filename string, data bytes.Buffer) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data.Bytes()); err != nil {
		return err
	}

	return nil
}
