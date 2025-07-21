package reports

import (
	"bytes"
	"sort"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
)

var summaryTmpl = template.Must(
	template.New("summary.tmpl").
		Funcs(template.FuncMap{
			"add":      add,
			"pad":      Pad,
			"padNum":   PadNum,
			"padFloat": PadFloat,
		}).
		ParseFS(tmplFS, "templates/summary.tmpl"),
)

func ExportDriverSummaries(store *database.Store, config *configuration.Config) error {
	var sums []database.DriverSummary

	sql := database.GetSeasonSummaryQuery(config)
	if err := store.DB.Raw(sql).Scan(&sums).Error; err != nil {
		return err
	}

	sort.Slice(sums, func(i, j int) bool {
		return sums[i].TotalChampionshipPoints > sums[j].TotalChampionshipPoints
	})

	var buf bytes.Buffer
	if err := summaryTmpl.Execute(&buf, sums); err != nil {
		return err
	}

	if err := writeMarkdown("driver_summaries.md", buf); err != nil {
		return err
	}
	return nil
}
