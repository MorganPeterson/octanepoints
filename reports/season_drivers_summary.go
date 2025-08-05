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

	// Export based on configured format
	switch config.Report.Format {
	case "markdown":
		return exportDriverSummariesMarkdown(sums, config)
	case "csv":
		return exportDriverSummariesCSV(sums, config)
	case "both":
		if err := exportDriverSummariesMarkdown(sums, config); err != nil {
			return err
		}
		return exportDriverSummariesCSV(sums, config)
	default:
		return fmt.Errorf("unsupported report format: %s", config.Report.Format)
	}
}

func exportDriverSummariesMarkdown(sums []database.DriverSummary, config *configuration.Config) error {
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

func exportDriverSummariesCSV(sums []database.DriverSummary, config *configuration.Config) error {
	// Create CSV records
	records := [][]string{}

	// Header
	records = append(records, []string{
		"Driver",
		"Nationality",
		"Rallies Started",
		"Rally Wins",
		"Podiums",
		"Stage Wins",
		"Best Position",
		"Average Position",
		"Total Super Rallied Stages",
		"Total Championship Points",
	})

	// Data rows
	for _, summary := range sums {
		records = append(records, []string{
			summary.UserName,
			summary.Nationality,
			fmt.Sprintf("%d", summary.RalliesStarted),
			fmt.Sprintf("%d", summary.RallyWins),
			fmt.Sprintf("%d", summary.Podiums),
			fmt.Sprintf("%d", summary.StageWins),
			fmt.Sprintf("%d", summary.BestPosition),
			fmt.Sprintf("%.2f", summary.AveragePosition),
			fmt.Sprintf("%d", summary.TotalSuperRalliedStages),
			fmt.Sprintf("%d", summary.TotalChampionshipPoints),
		})
	}

	// create file name and write CSV
	fileName := fmt.Sprintf("%s.%s", config.Report.Drivers.SeasonSummaryFilename, "csv")

	return writeCSV(fileName, records, config)
}
