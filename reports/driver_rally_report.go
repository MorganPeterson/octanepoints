package reports

import (
	"bytes"
	"fmt"
	"log"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
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
					return Pad("-", 12)
				}
				return Pad(fmt.Sprintf("+%.3f s", d), 12)
			},
			"formatPenalty": func(p float64) string {
				return Pad(fmt.Sprintf("%.0f", p), 3)
			},
			"add":    add,
			"pad":    Pad,
			"padNum": PadNum,
		}).
		ParseFS(tmplFS, "templates/driver_summary.tmpl"),
)

func DriverRallyReport(rallyIdStr string, store *database.Store, config *configuration.Config) error {
	// Parse the rally ID from the command line argument
	rallyId := database.ParseStringToUint(rallyIdStr)

	// Get stages summary for the rally
	summaries, err := stagesSummary(rallyId, store)
	if err != nil {
		log.Fatalf("Failed to get stages summary: %v", err)
	}

	var buf bytes.Buffer
	if err := driverSummary.Execute(&buf, summaries); err != nil {
		return err
	}

	if err := writeMarkdown("driver_summary.md", buf); err != nil {
		return err
	}
	return nil
}

func stagesSummary(
	rallyId uint64, store *database.Store,
) (map[string]DriverReport, error) {
	var userNames []string
	err := store.DB.Model(&database.RallyStage{}).
		Where("rally_id = ?", rallyId).
		Distinct("user_name").
		Pluck("user_name", &userNames).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user names: %w", err)
	}

	log.Printf("Found %d drivers for rally %d", len(userNames), rallyId)
	summary := make(map[string]DriverReport)
	for _, userName := range userNames {
		var stages []StageSummary
		sql := database.DriverStagesQuery()
		err = store.DB.Raw(sql, rallyId, userName).Scan(&stages).Error
		if err != nil {
			return nil, fmt.Errorf("failed to fetch stages summary for %s: %w", userName, err)
		}

		if len(stages) == 0 {
			continue // skip drivers with no stages
		}

		var overall []SummaryRow
		overall, err = getDriverOverallSummary(rallyId, userName, store)
		if err != nil {
			log.Printf("failed to fetch overall summary for %s: %v", userName, err)
		}

		summary[userName] = DriverReport{
			Stages:  stages,
			Overall: overall,
		}
	}

	return summary, nil
}
