package reports

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
	"git.sr.ht/~nullevoid/octanepoints/parser"
)

var driverSummary = template.Must(
	template.New("driver_summary.tmpl").
		Funcs(template.FuncMap{
			"formatStageTime": func(sec float64) string {
				min := int(sec) / 60
				s := sec - float64(min*60)
				return fmt.Sprintf("%02d:%06.3f", min, s)
			},
			"formatDelta": func(d float64) string {
				if d == 0 {
					return pad("-", 12)
				}
				return pad(fmt.Sprintf("+%.3f s", d), 12)
			},
			"formatPenalty": func(p float64) string {
				return pad(fmt.Sprintf("%.0f", p), 3)
			},
			"add":    add,
			"pad":    pad,
			"padNum": padNum,
		}).
		ParseFS(tmplFS, "templates/driver_summary.tmpl"),
)

type SummaryRow struct {
	Metric      string
	DriverValue string
	FieldAvg    string
	RankText    string
}

type DriverReport struct {
	Stages  []database.StageSummary
	Overall []SummaryRow
}

type DriverReportConfig struct {
	Overall   []database.RallyOverall
	Finishers []database.RallyOverall
	Config    RallyConfig
}

type RallyConfig struct {
	WinnerTime   time.Duration
	TotalDrivers int
	AvgTime      time.Duration
	AvgPenalty   float64
	AvgSuper     float64
}

func DriverRallyReport(rallyId int64, store *database.Store, config *configuration.Config) error {
	// Get stages summary for the rally
	summaries, err := stagesSummary(rallyId, store)
	if err != nil {
		return fmt.Errorf("Failed to get stages summary: %v", err)
	}

	// Export based on configured format
	switch config.Report.Format {
	case "markdown":
		return exportDriverRallyMarkdown(rallyId, summaries, config)
	case "csv":
		return exportDriverRallyCSV(rallyId, summaries, config)
	case "both":
		if err := exportDriverRallyMarkdown(rallyId, summaries, config); err != nil {
			return err
		}
		return exportDriverRallyCSV(rallyId, summaries, config)
	default:
		return fmt.Errorf("unsupported report format: %s", config.Report.Format)
	}
}

func exportDriverRallyMarkdown(rallyId int64, summaries map[string]DriverReport, config *configuration.Config) error {
	var buf bytes.Buffer
	if err := driverSummary.Execute(&buf, summaries); err != nil {
		return err
	}
	// create file name and write markdown
	fileName := fmt.Sprintf("%d_%s.%s", rallyId, config.Report.Drivers.RallySummaryFilename, "md")

	if err := writeMarkdown(fileName, buf, config); err != nil {
		return err
	}
	return nil
}

func exportDriverRallyCSV(rallyId int64, summaries map[string]DriverReport, config *configuration.Config) error {
	// Create CSV records
	records := [][]string{}

	// Header
	records = append(records, []string{fmt.Sprintf("Rally %d - Driver Summaries", rallyId)})
	records = append(records, []string{})

	// For each driver
	for driverName, report := range summaries {
		records = append(records, []string{fmt.Sprintf("Driver: %s", driverName)})
		records = append(records, []string{})

		// Stages section
		if len(report.Stages) > 0 {
			records = append(records, []string{"Stage Results"})
			records = append(records, []string{"Stage", "Time", "Position", "Delta to Winner", "Penalty", "Comments"})

			for _, stage := range report.Stages {
				records = append(records, []string{
					stage.StageName,
					fmt.Sprintf("%.3f", stage.StageTime),
					fmt.Sprintf("%d", stage.Position),
					fmt.Sprintf("%.3f", stage.DeltaToWinner),
					fmt.Sprintf("%.0f", stage.Penalty),
					stage.Comments,
				})
			}
			records = append(records, []string{})
		}

		// Overall summary section
		if len(report.Overall) > 0 {
			records = append(records, []string{"Overall Summary"})
			records = append(records, []string{"Metric", "Value", "Field Average", "Rank"})

			for _, overall := range report.Overall {
				records = append(records, []string{
					overall.Metric,
					overall.DriverValue,
					overall.FieldAvg,
					overall.RankText,
				})
			}
		}
		records = append(records, []string{})
		records = append(records, []string{})
	}

	// create file name and write CSV
	fileName := fmt.Sprintf("%d_%s.%s", rallyId, config.Report.Drivers.RallySummaryFilename, "csv")

	return writeCSV(fileName, records, config)
}

func configSummaries(rallyId int64, store *database.Store) (DriverReportConfig, error) {
	overall, err := database.GetRallyOverall(store, &database.QueryOpts{RallyId: &rallyId})
	if err != nil {
		return DriverReportConfig{}, fmt.Errorf("no overall summary error=%v", err)
	}

	finishers, err := database.GetDriversRallySummary(store, &database.QueryOpts{RallyId: &rallyId})
	if err != nil {
		return DriverReportConfig{}, fmt.Errorf("no finishers error=%v", err)
	}

	if len(finishers) == 0 {
		return DriverReportConfig{}, fmt.Errorf("no finishers found for rally %d", rallyId)
	}

	totalDrivers := len(finishers)
	winnerTime := finishers[0].Time3

	var sumTime time.Duration
	var sumPenalty float64
	var sumSuper int64
	for _, r := range overall {
		sumTime += r.Time3
		sumPenalty += r.Penalty
		sumSuper += r.SuperRally
	}

	avgTime := time.Duration(int64(sumTime) / int64(totalDrivers))
	avgPenalty := sumPenalty / float64(totalDrivers)
	avgSuper := float64(sumSuper) / float64(totalDrivers)

	return DriverReportConfig{
		Overall:   overall,
		Finishers: finishers,
		Config: RallyConfig{
			WinnerTime:   winnerTime,
			TotalDrivers: totalDrivers,
			AvgTime:      avgTime,
			AvgPenalty:   avgPenalty,
			AvgSuper:     avgSuper,
		},
	}, nil
}

func getDriverOverallSummary(name string, config DriverReportConfig) ([]SummaryRow, error) {
	var driver *database.RallyOverall
	var finishRank int
	for i, r := range config.Finishers {
		if r.UserName == name {
			driver = &r
			finishRank = i + 1
			break
		}
	}

	if driver == nil {
		// they have dnf'd; look at full list
		for i, r := range config.Overall {
			if r.UserName == name {
				driver = &r
				finishRank = i + 1
				break
			}
		}
	}
	rows := []SummaryRow{
		{
			Metric: "Finishing Position",
			DriverValue: func() string {
				if driver == nil || driver.Time3 == 0 {
					return "DNF"
				}
				return fmt.Sprintf("%d", finishRank)
			}(),
			FieldAvg: "-",
			RankText: func() string {
				if driver == nil || driver.Time3 == 0 {
					return ""
				}
				return fmt.Sprintf("%d/%d", finishRank, config.Config.TotalDrivers)
			}(),
		},
		{
			Metric: "Total Time",
			DriverValue: func() string {
				if driver == nil || driver.Time3 == 0 {
					return "DNF"
				}
				return parser.FmtDuration(driver.Time3)
			}(),
			FieldAvg: parser.FmtDuration(config.Config.AvgTime),
			RankText: "",
		},
		{
			Metric: "Delta to Winner",
			DriverValue: func() string {
				if driver == nil || driver.Time3 == 0 {
					return "DNF"
				}
				delta := driver.Time3 - config.Config.WinnerTime
				return parser.FmtDuration(delta)
			}(),
			FieldAvg: "-",
			RankText: "",
		},
		{
			Metric: "Total Penalty",
			DriverValue: func() string {
				if driver == nil {
					return "DNF"
				}
				return fmt.Sprintf("%.0fs", driver.Penalty)
			}(),
			FieldAvg: fmt.Sprintf("%.1fs", config.Config.AvgPenalty),
			RankText: "",
		},
		{
			Metric: "Super Rallies",
			DriverValue: func() string {
				if driver == nil {
					return "DNF"
				}
				return fmt.Sprintf("%d", driver.SuperRally)
			}(),

			FieldAvg: fmt.Sprintf("%.1f", config.Config.AvgSuper),
			RankText: "",
		},
	}

	return rows, nil
}

func stagesSummary(rallyId int64, store *database.Store) (map[string]DriverReport, error) {
	userNames, err := database.GetRallyUserNames(store, rallyId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user names: %w", err)
	}

	summary := make(map[string]DriverReport)
	driverConfig, err := configSummaries(rallyId, store)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver report config: %w", err)
	}

	for _, userName := range userNames {
		stages, err := database.GetDriverStages(store, rallyId, userName)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch stages for %s: %w", userName, err)
		}

		if len(stages) == 0 {
			continue // skip drivers with no stages
		}

		var overall []SummaryRow
		overall, err = getDriverOverallSummary(userName, driverConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch overall summary for %s: %v", userName, err)
		}

		summary[userName] = DriverReport{
			Stages:  stages,
			Overall: overall,
		}
	}

	return summary, nil
}
