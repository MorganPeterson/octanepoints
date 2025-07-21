package reports

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strconv"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
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
	UserId   uint64
	UserName string
	Points   int64
}

var reportTmpl = template.Must(
	template.New("report.tmpl").
		Funcs(template.FuncMap{
			"add":      add,
			"pad":      Pad,
			"padNum":   PadNum,
			"padFloat": PadFloat,
		}).
		ParseFS(tmplFS, "templates/report.tmpl"),
)

func ExportReport(rallyIdStr string, store *database.Store, config *configuration.Config) error {
	// Parse the rally ID from the command line argument
	rallyId := database.ParseStringToUint(rallyIdStr)

	// Assign points to the overall results
	scored, err := assignPointsOverall(rallyId, store, config)
	if err != nil {
		log.Fatalf("Failed to assign points: %v", err)
	}

	// Fetch championship points
	standings, err := fetchChampionshipPoints(store, config)
	if err != nil {
		log.Fatalf("Failed to fetch championship points: %v", err)
	}

	// Prepare report data
	data := ReportData{
		Rally:        scored,
		Championship: standings,
	}

	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, data); err != nil {
		return err
	}

	if err := writeMarkdown("report.md", buf); err != nil {
		return err
	}

	return nil
}

// assignPointsOverall assigns points to each record based on the configured points system.
func assignPointsOverall(
	rallyId uint64, store *database.Store, config *configuration.Config,
) ([]ScoreRecord, error) {
	// Fetch the overall results from the database
	overallData, err := database.GetRallyOverall(rallyId, store)
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
	recs, err := database.GetAllRallyOveralls(store)
	if err != nil {
		return nil, fmt.Errorf("fetching overall records: %w", err)
	}

	standingsMap := make(map[uint64]*SeasonsStandings)
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
