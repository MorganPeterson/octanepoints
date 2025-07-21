package reports

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"strings"

	"github.com/raykov/mdtopdf"
)

//go:embed templates/*.tmpl
var tmplFS embed.FS

func add(a, b int) int { return a + b }

// pad right-spaces a string to width w
func Pad(s string, w int) string {
	if len(s) >= w {
		return s[:w]
	}
	return s + string(bytes.Repeat([]byte(" "), w-len(s)))
}

func PadNum(n int64, w int) string {
	s := fmt.Sprint(n)
	if len(s) >= w {
		return s
	}
	return string(bytes.Repeat([]byte(" "), w-len(s))) + s
}

func PadFloat(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return string(bytes.Repeat([]byte(" "), w-len(s))) + s
}

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
