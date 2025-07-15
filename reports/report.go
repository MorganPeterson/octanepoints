package reports

import (
	"os"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/database"
	"git.sr.ht/~nullevoid/octanepoints/points"
)

type ReportData struct {
	Rally        []points.ScoreRecord
	Championship []points.SeasonsStandings
}

func add(a, b int) int { return a + b }

func ExportMarkdown(filename string, data ReportData) error {
	tpl := template.Must(
		template.New("").
			Funcs(template.FuncMap{"add": add}).
			ParseFiles("reports/templates/report.tmpl"),
	)

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// this will execute the single, top‑level template
	return tpl.ExecuteTemplate(f, "report.tmpl", data)
}

func ExportDriverSummaries(filename string, sums []database.DriverSummary) error {
	tpl, err := template.ParseFiles("reports/templates/summary.tmpl")
	if err != nil {
		return err
	}

	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return tpl.ExecuteTemplate(f, "summary.tmpl", sums)
}
