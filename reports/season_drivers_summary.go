package reports

import (
	"bytes"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
)

var summaryTmpl = template.Must(
	template.New("summary.tmpl").
		Funcs(template.FuncMap{
			"add":      add,
			"pad":      pad,
			"padNum":   padNum,
			"padFloat": padFloat,
		}).
		ParseFS(tmplFS, "templates/summary.tmpl"),
)

func ExportDriverSummaries(store *database.Store, config *configuration.Config) error {
	var sums []database.DriverSummary

	sums, err := database.GetSeasonSummary(store, config)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := summaryTmpl.Execute(&buf, sums); err != nil {
		return err
	}

	if err := writeMarkdown("driver_summaries.md", buf); err != nil {
		return err
	}
	return nil
}
