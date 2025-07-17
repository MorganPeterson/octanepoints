package reports

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/database"
	"git.sr.ht/~nullevoid/octanepoints/points"
)

type ReportData struct {
	Rally        []points.ScoreRecord
	Championship []points.SeasonsStandings
}

//go:embed templates/*.tmpl
var tmplFS embed.FS

var (
	summaryTmpl = template.Must(
		template.New("summary.tmpl").
			Funcs(template.FuncMap{
				"add":      add,
				"pad":      Pad,
				"padNum":   PadNum,
				"padFloat": PadFloat,
			}).
			ParseFS(tmplFS, "templates/summary.tmpl"),
	)

	reportTmpl = template.Must(
		template.New("report.tmpl").
			Funcs(template.FuncMap{
				"add":      add,
				"pad":      Pad,
				"padNum":   PadNum,
				"padFloat": PadFloat,
			}).
			ParseFS(tmplFS, "templates/report.tmpl"),
	)
)

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

func ExportReport(filename string, data ReportData) error {
	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func ExportDriverSummaries(filename string, sums []database.DriverSummary) error {
	var buf bytes.Buffer
	if err := summaryTmpl.Execute(&buf, sums); err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}
