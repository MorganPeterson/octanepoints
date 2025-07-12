package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
)

// #;userid;user_name;real_name;nationality;car;time3;super_rally;penalty
type OverallRow struct {
	Position    string
	UserId      int64
	UserName    string
	RealName    string
	Nationality string
	Car         string
	Time3       string
	SuperRally  bool
	Penalty     *float64
}

// ScoreRecord holds the raw data and the assigned points for each record.
type ScoreRecord struct {
	Raw    OverallRow
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
	rallyId := parseStringToInt64(args[0])

	// Load the configuration
	config := configuration.MustLoad(configPath)

	// Fetch the CSV data and assign points
	scored, err := fetchCSV(rallyId, config)
	if err != nil {
		log.Fatal(err)
	}

	// must print the headers first
	fmt.Println(config.Headers)

	// Print the scored records
	for _, rec := range scored {
		fmt.Printf("%s,%s,%d\n", rec.Raw.Position, rec.Raw.UserName, rec.Points)
	}
}

// fetchCSV retrieves the CSV data from the configured URL and parses it into ScoreRecords.
func fetchCSV(rallyId int64, config *configuration.Config) ([]ScoreRecord, error) {
	fetchUrl := fmt.Sprintf(config.URL, rallyId, config.CG)

	// build client
	client := &http.Client{Timeout: 10 * time.Second} // 10 seconds timeout
	req, _ := http.NewRequest("GET", fetchUrl, nil)
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

	// If you have a header row and want to skip it:
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	var out []OverallRow
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading row: %w", err)
		}
		// #;userid;user_name;real_name;nationality;car;time3;super_rally;penalty
		rec := OverallRow{
			Position:    row[0],
			UserId:      parseStringToInt64(row[1]),
			UserName:    row[2],
			RealName:    row[3],
			Nationality: row[4],
			Car:         row[5],
			Time3:       row[6],
			SuperRally:  parseStringToBool(row[7]),
			Penalty:     parseStringToFloat(row[7]),
		}
		out = append(out, rec)
	}

	return assignPointsOverall(out, config)
}

// assignPointsOverall assigns points to each record based on the configured points system.
func assignPointsOverall(recs []OverallRow, config *configuration.Config) ([]ScoreRecord, error) {
	scored := make([]ScoreRecord, len(recs))
	for i, rec := range recs {
		pts := 0
		if i < len(config.Points) {
			pts = config.Points[i]
		}
		scored[i] = ScoreRecord{
			Raw:    rec,
			Points: pts,
		}
	}
	return scored, nil
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

func parseStringToFloat(s string) *float64 {
	if s == "" {
		return nil
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &value
}
