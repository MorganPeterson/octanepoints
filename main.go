package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net/http"
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

const (
	configPath     string = "config.toml"                    // Path to the configuration file
	ErrorNoRallyId string = "Error: no rally ID provided"    // Error message when no rally ID is given
	UsageString    string = "Usage: octanepoints <rally_id>" // Usage string for the command
)

func main() {
	flag.Parse()

	// rallyId is expected
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, ErrorNoRallyId)
		fmt.Fprintln(os.Stderr, UsageString)
		os.Exit(1)
	}

	// Parse the rally ID from the command line argument
	rallyId := parseStringToUint(args[0])

	// Load the configuration
	config := configuration.MustLoad(configPath)

	// init database store
	store, err := database.NewStore(config.Database.Name)
	if err != nil {
		log.Fatalf("Failed to initialize database store: %v", err)
	}
	defer store.Close()

	var overallData []database.RallyOverall
	overallData, err = fetchRallyOverallDB(rallyId, store)
	if err != nil {
		overallData, err = fetchOverall(rallyId, store, config)
		if err != nil {
			log.Fatalf("Failed to fetch rally data: %v", err)
		}
	}

	_, err = fetchRallyStagesDB(rallyId, store)
	if err != nil {
		_, err = fetchStages(rallyId, store, config)
		if err != nil {
			log.Fatalf("Failed to fetch stages data: %v", err)
		}
	}

	// Assign points
	scored, err := assignPointsOverall(overallData, config)
	if err != nil {
		log.Fatalf("Failed to assign points: %v", err)
	}

	// must print the headers first
	fmt.Println(config.General.Headers)

	// Print the scored records
	for _, rec := range scored {
		fmt.Printf("%s,%s,%d\n", rec.Raw.Position, rec.Raw.UserName, rec.Points)
	}
}

func fetchRallyOverallDB(
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

func fetchRallyStagesDB(
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

func fetchCsv(url string) ([][]string, error) {
	// build client
	client := &http.Client{Timeout: 10 * time.Second} // 10 seconds timeout
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching CSV: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching CSV: received status code %d", resp.StatusCode)
	}

	r := csv.NewReader(resp.Body)
	r.Comma = ';'

	// Read all records
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("reading CSV: %w", err)
	}

	return records, nil
}

// fetchCSV retrieves the CSV data from the configured URL and parses it into ScoreRecords.
func fetchOverall(
	rallyId uint64, store *database.Store, config *configuration.Config,
) ([]database.RallyOverall, error) {
	// build the URL to fetch the CSV data
	fetchUrl := fmt.Sprintf(config.HTTP.OverallURL, rallyId, config.HTTP.CG)

	r, err := fetchCsv(fetchUrl)
	if err != nil {
		return nil, fmt.Errorf("fetching CSV from %s: %w", fetchUrl, err)
	}

	var out []database.RallyOverall
	for _, row := range r[1:] { // skip header row
		// #;userid;user_name;real_name;nationality;car;time3;super_rally;penalty
		rec := database.RallyOverall{
			RallyId:     rallyId,
			UserId:      parseStringToUint(row[1]),
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
			return nil, fmt.Errorf("storing record in database: %w", err)
		}

		out = append(out, rec)
	}

	return out, nil
}

// assignPointsOverall assigns points to each record based on the configured points system.
func assignPointsOverall(
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

// fetchCSV retrieves the CSV data from the configured URL and parses it into ScoreRecords.
func fetchStages(
	rallyId uint64, store *database.Store, config *configuration.Config,
) ([]database.RallyStage, error) {
	// build the URL to fetch the CSV data
	fetchUrl := fmt.Sprintf(config.HTTP.BetaUrl, rallyId)

	r, err := fetchCsv(fetchUrl)
	if err != nil {
		return nil, fmt.Errorf("fetching CSV from %s: %w", fetchUrl, err)
	}

	var out []database.RallyStage
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
			return nil, fmt.Errorf("storing record in database: %w", err)
		}

		out = append(out, rec)
	}

	return out, nil
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

func parseStringToUint(s string) uint64 {
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
			log.Printf("invalid minutes: %w", err)
			return 0
		}
		secF, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Printf("invalid seconds: %w", err)
			return 0
		}

	case 3:
		// HH:MM:SS.sss
		h, err = strconv.Atoi(parts[0])
		if err != nil {
			log.Printf("invalid hours: %w", err)
			return 0
		}
		m, err = strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("invalid minutes: %w", err)
			return 0
		}
		secF, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			log.Printf("invalid seconds: %w", err)
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
