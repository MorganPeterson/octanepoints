package points

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
)

// ScoreRecord holds the raw data and the assigned points for each record.
type ScoreRecord struct {
	Raw    database.RallyOverall
	Points int
}

// assignPointsOverall assigns points to each record based on the configured points system.
func AssignPointsOverall(
	recs []database.RallyOverall, config *configuration.Config,
) ([]ScoreRecord, error) {
	scored := make([]ScoreRecord, len(recs))
	for i, rec := range recs {
		pts := 0
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

func FetchRallyOverallDB(
	rallyId uint64, store *database.Store,
) ([]database.RallyOverall, error) {
	// check if the overall is already in database
	var existing []database.RallyOverall
	err := store.DB.Where("rally_id = ?", rallyId).Find(&existing).Error
	if err == nil && len(existing) > 0 {
		return existing, nil
	}

	return nil, fmt.Errorf("no records found for rally ID %d", rallyId)
}

func FetchRallyStagesDB(
	rallyId uint64, store *database.Store,
) ([]database.RallyStage, error) {
	// check if the stages are already in database
	var existing []database.RallyStage
	err := store.DB.Where("rally_id = ?", rallyId).Find(&existing).Error
	if err == nil && len(existing) > 0 {
		return existing, nil
	}

	return nil, fmt.Errorf("no records found for rally ID %d", rallyId)
}

func ParseStringToUint(s string) uint64 {
	if s == "" {
		return 0
	}
	value, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing uint64: %v\n", err)
		return 0
	}
	return value
}

// StoreOverall stores the overall results from the CSV file into the database.
func StoreOverall(rallyId uint64, store *database.Store, config *configuration.Config) error {
	csvPath := fmt.Sprintf("%d/%d_All_table.csv", rallyId, rallyId)
	r, err := fetchCsv(csvPath, config)
	if err != nil {
		return err
	}

	for _, row := range r[1:] { // skip header row
		// #;userid;user_name;real_name;nationality;car;time3;super_rally;penalty
		rec := database.RallyOverall{
			RallyId:     rallyId,
			UserId:      ParseStringToUint(row[1]),
			Position:    row[0],
			UserName:    row[2],
			RealName:    row[3],
			Nationality: row[4],
			Car:         row[5],
			Time3:       parseHMS(row[6]),
			SuperRally:  parseStringToInt64(row[7]),
			Penalty:     parseStringToFloat(row[8]),
		}

		if err := store.DB.Create(&rec).Error; err != nil {
			return fmt.Errorf("storing record in database: %w", err)
		}
	}
	return nil
}

func StoreRally(rallyId uint64, config *configuration.Config, store *database.Store) error {
	rallyPath := fmt.Sprintf("%s/%d/%d.toml", config.General.DescriptionDir, rallyId, rallyId)
	rally, err := configuration.LoadRally(rallyPath)
	if err != nil {
		return fmt.Errorf("loading rally description: %w", err)
	}

	// Store the rally information in the database
	if err := store.DB.Create(rally).Error; err != nil {
		return fmt.Errorf("storing rally in database: %w", err)
	}

	return nil
}

// StoreStages stores the stages from the CSV file into the database.
func StoreStages(rallyId uint64, store *database.Store, config *configuration.Config) error {
	csvPath := fmt.Sprintf("%d/%d_table.csv", rallyId, rallyId)
	r, err := fetchCsv(csvPath, config)
	if err != nil {
		return err
	}

	for _, row := range r[1:] { // skip header row
		rec := database.RallyStage{
			RallyId:        rallyId,
			StageNum:       parseStringToInt64(row[0]),
			StageName:      row[1],
			Nationality:    row[2],
			UserName:       row[3],
			RealName:       row[4],
			Group:          row[5],
			CarName:        row[6],
			Time1:          parseStringToFloat(row[7]),
			Time2:          parseStringToFloat(row[8]),
			Time3:          parseStringToFloat(row[9]),
			FinishRealTime: parseFinishRealTime(row),
			Penalty:        parseStringToFloat(row[11]),
			ServicePenalty: parseStringToFloat(row[12]),
			SuperRally:     parseStringToBool(row[13]),
			Progress:       row[14],
			Comments:       row[15],
		}

		if err := store.DB.Create(&rec).Error; err != nil {
			return fmt.Errorf("storing record in database: %w", err)
		}
	}

	return nil
}

func fetchCsv(path string, config *configuration.Config) ([][]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}
	filePath := fmt.Sprintf("%s/%s/%s", currentDir, config.General.DescriptionDir, path)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening CSV file %s: %w", filePath, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = ';'

	return reader.ReadAll()
}

// FormatHMS renders a time.Duration as either "MM:SS.sss" (if <1h)
// or "HH:MM:SS.sss" when hours are non-zero.
func formatHMS(d time.Duration) string {
	h := d / time.Hour
	rem := d % time.Hour
	m := rem / time.Minute
	s := rem % time.Minute

	if h > 0 {
		// include hours
		return fmt.Sprintf("%02d:%02d:%06.3f", h, m, s.Seconds())
	}
	// no hours
	return fmt.Sprintf("%02d:%06.3f", m, s.Seconds())
}

func parseFinishRealTime(row []string) time.Time {
	FinishRealTime, err := time.Parse("2006-01-02 15:04:05", row[10])
	if err != nil {
		log.Printf("Error parsing FinishRealTime: %v", err)
		return time.Time{}
	}
	return FinishRealTime
}

// ParseHMS parses a string in "MM:SS.sss" or "HH:MM:SS.sss" format into a time.Duration.
// It returns an error if the format is invalid.
func parseHMS(s string) time.Duration {
	parts := strings.Split(s, ":")
	var (
		h, m int
		secF float64
		err  error
	)

	switch len(parts) {
	case 2:
		// MM:SS.sss
		m, err = strconv.Atoi(parts[0])
		if err != nil {
			log.Printf("invalid minutes: %+v", err)
			return 0
		}
		secF, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Printf("invalid seconds: %+v", err)
			return 0
		}

	case 3:
		// HH:MM:SS.sss
		h, err = strconv.Atoi(parts[0])
		if err != nil {
			log.Printf("invalid hours: %v", err)
			return 0
		}
		m, err = strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("invalid minutes: %+v", err)
			return 0
		}
		secF, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			log.Printf("invalid seconds: %+v", err)
			return 0
		}

	default:
		log.Printf("invalid time format %q", s)
		return 0
	}

	// build duration
	return time.Duration(h)*time.Hour +
		time.Duration(m)*time.Minute +
		time.Duration(secF*float64(time.Second))
}

func parseStringToBool(s string) bool {
	var value bool
	switch s {
	case "1":
		value = true
	default:
		value = false
	}
	return value
}

func parseStringToFloat(s string) float64 {
	if s == "" {
		return 0
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseStringToInt64(s string) int64 {
	if s == "" {
		return 0
	}
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing int64: %v\n", err)
		return 0
	}
	return value
}
