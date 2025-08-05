package reports

import (
	"bytes"
	"embed"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/parser"
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

func writeMarkdown(filename string, data bytes.Buffer, config *configuration.Config) error {
	// get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}
	reportPath := filepath.Join(
		currentDir,
		config.General.Directory,
		config.Report.Directory,
		config.Report.MdDirectory,
		filename,
	)
	f, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(data.Bytes()); err != nil {
		return err
	}

	return nil
}

func writeCSV(filename string, records [][]string, config *configuration.Config) error {
	// get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}
	reportPath := filepath.Join(
		currentDir,
		config.General.Directory,
		config.Report.Directory,
		config.Report.CsvDirectory,
		filename,
	)
	f, err := os.Create(reportPath)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.Comma = rune(config.Report.Delimiter[0])
	defer writer.Flush()

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("writing CSV record: %w", err)
		}
	}

	return nil
}
