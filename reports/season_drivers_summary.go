package reports

import (
	"bytes"
	"fmt"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
)

var summaryTmpl = template.Must(
	template.New("summary.tmpl").
		Funcs(sharedFuncMap).
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

	// create file name and write markdown
	fileName := fmt.Sprintf("%s.%s", config.Report.Drivers.SeasonSummaryFilename, "md")

	if err := writeMarkdown(fileName, buf, config); err != nil {
		return err
	}

	return nil
}
