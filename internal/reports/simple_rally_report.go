package reports

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"text/template"

	"github.com/MorganPeterson/octanepoints/internal/configuration"
	"github.com/MorganPeterson/octanepoints/internal/database"
)

type ReportData struct {
	Rally        []ScoreRecord
	Championship []SeasonsStandings
}

// ScoreRecord holds the raw data and the assigned points for each record.
type ScoreRecord struct {
	Raw    database.RallyOverall
	Points int64
}

type SeasonsStandings struct {
	UserId   int64
	UserName string
	Points   int64
}

var reportTmpl = template.Must(
	template.New("report.tmpl").
		Funcs(sharedFuncMap).
		ParseFS(tmplFS, "templates/report.tmpl"),
)

func ExportReport(rallyId int64, store *database.Store, config *configuration.Config) error {
	// Assign points to the overall results
	scored, err := assignPointsOverall(rallyId, store, config)
	if err != nil {
		return fmt.Errorf("Failed to assign points: %v", err)
	}

	// Fetch championship points
	standings, err := fetchChampionshipPoints(store, config)
	if err != nil {
		return fmt.Errorf("Failed to fetch championship points: %v", err)
	}

	// Prepare report data
	data := ReportData{
		Rally:        scored,
		Championship: standings,
	}

	// Export based on configured format
	switch config.Report.Format {
	case "markdown":
		return exportMarkdown(rallyId, data, config)
	case "csv":
		return exportCSV(rallyId, data, config)
	case "both":
		if err := exportMarkdown(rallyId, data, config); err != nil {
			return err
		}
		return exportCSV(rallyId, data, config)
	default:
		return fmt.Errorf("unsupported report format: %s", config.Report.Format)
	}
}

func exportMarkdown(rallyId int64, data ReportData, config *configuration.Config) error {
	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		return err
	}

	// create file name and write markdown
	fileName := fmt.Sprintf("%d_%s.%s", rallyId, config.Report.Points.SummaryFileName, "md")

	if err := writeMarkdown(fileName, buf, config); err != nil {
		return err
	}

	return nil
}

func exportCSV(rallyId int64, data ReportData, config *configuration.Config) error {
	// Create CSV records
	rallyResults := [][]string{}

	// Rally results header
	rallyResults = append(rallyResults, []string{"Rally Id", "Position", "Driver", "Car", "Time", "Points"})

	// Rally results data
	for i, record := range data.Rally {
		position := fmt.Sprintf("%d", i+1)
		rallyResults = append(rallyResults, []string{
			strconv.FormatInt(rallyId, 10),
			position,
			record.Raw.UserName,
			record.Raw.Car,
			record.Raw.Time3.String(),
			fmt.Sprintf("%d", record.Points),
		})
	}

	err := writeCSV(fmt.Sprintf("%d_%s.%s", rallyId, "rally_overall_points", "csv"), rallyResults, config)
	if err != nil {
		return fmt.Errorf("Failed to write CSV: %v", err)
	}

	overallChampionship := [][]string{}
	// Championship standings header
	overallChampionship = append(overallChampionship, []string{"Rally Id", "Position", "Driver", "Total Points"})

	// Championship standings data
	for i, standing := range data.Championship {
		position := fmt.Sprintf("%d", i+1)
		overallChampionship = append(overallChampionship, []string{
			fmt.Sprintf("%d", rallyId),
			position,
			standing.UserName,
			fmt.Sprintf("%d", standing.Points),
		})
	}

	// create file name and write CSV
	fileName := fmt.Sprintf("%d_%s.%s", rallyId, "championship_standings", "csv")

	return writeCSV(fileName, overallChampionship, config)
}

// assignPointsOverall assigns points to each record based on the configured points system.
func assignPointsOverall(
	rallyId int64, store *database.Store, config *configuration.Config,
) ([]ScoreRecord, error) {
	// Fetch the overall results from the database
	overallData, err := database.GetRallyOverall(store, &database.QueryOpts{RallyId: &rallyId})
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch overall data: %w", err)
	}

	scored := make([]ScoreRecord, len(overallData))
	for i, rec := range overallData {
		pts := int64(0)
		if i < len(config.General.Points) {
			pts = config.General.Points[i]
		}
		scored[i] = ScoreRecord{
			Raw:    rec,
			Points: pts,
		}
	}
	return scored, nil
}

func fetchChampionshipPoints(
	store *database.Store, config *configuration.Config,
) ([]SeasonsStandings, error) {
	recs, err := database.GetRallyOverall(store, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching overall records: %w", err)
	}

	standingsMap := make(map[int64]*SeasonsStandings)
	for _, r := range recs {
		pos, err := strconv.Atoi(r.Position)
		if err != nil || pos < 1 || pos > len(config.General.Points) {
			continue
		}
		pts := config.General.Points[pos-1]
		if entry, ok := standingsMap[r.UserId]; ok {
			entry.Points += int64(pts)
		} else {
			standingsMap[r.UserId] = &SeasonsStandings{
				UserId:   r.UserId,
				UserName: r.UserName,
				Points:   int64(pts),
			}
		}
	}

	standings := make([]SeasonsStandings, 0, len(standingsMap))
	for _, e := range standingsMap {
		standings = append(standings, *e)
	}

	// Sort standings by points in descending order
	sort.Slice(standings, func(i, j int) bool {
		return standings[i].Points > standings[j].Points
	})

	return standings, nil
}
