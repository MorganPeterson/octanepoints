package main

import (
	"flag"
	"fmt"
	"log"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
	"git.sr.ht/~nullevoid/octanepoints/points"
	"git.sr.ht/~nullevoid/octanepoints/reports"
)

const (
	configPath     string = "config.toml"                    // Path to the configuration file
	ErrorNoRallyId string = "Error: no rally ID provided"    // Error message when no rally ID is given
	UsageString    string = "Usage: octanepoints <rally_id>" // Usage string for the command
)

func main() {
	// flags
	createRallyFlag := flag.String("create", "", "create rally with given ID number")
	pointsRallyFlag := flag.String("points", "", "fetch points for a single rally given ID number")
	championshipFlag := flag.Bool("championship", false, "fetch championship points")
	reportFlag := flag.String("report", "", "export report to markdown file")

	flag.Parse()

	// Load the configuration
	config := configuration.MustLoad(configPath)

	// init database store
	store, err := database.NewStore(config.Database.Name)
	if err != nil {
		log.Fatalf("Failed to initialize database store: %v", err)
	}
	defer store.Close()

	if createRallyFlag != nil && *createRallyFlag != "" {
		// Parse the rally ID from the command line argument
		rallyId := points.ParseStringToUint(*createRallyFlag)
		// Create a new rally in the database
		if err := createRally(rallyId, config, store); err != nil {
			log.Fatalf("Failed to create rally: %v", err)
		}
		fmt.Printf("Rally %d created successfully.\n", rallyId)
		return
	}

	if pointsRallyFlag != nil && *pointsRallyFlag != "" {
		// Parse the rally ID from the command line argument
		rallyId := points.ParseStringToUint(*pointsRallyFlag)

		// Get points for the rally
		scored, err := getRallyPoints(rallyId, store, config)
		if err != nil {
			log.Fatalf("Failed to get rally points: %v", err)
		}

		// must print the headers first
		fmt.Println(config.General.Headers)

		// Print the scored records
		for _, rec := range scored {
			fmt.Printf("%s,%s,%d\n", rec.Raw.Position, rec.Raw.UserName, rec.Points)
		}

		return
	}

	if championshipFlag != nil && *championshipFlag {
		// Fetch championship points
		standings, err := points.FetchChampionshipPoints(store, config)
		if err != nil {
			log.Fatalf("Failed to fetch championship points: %v", err)
		}

		// Print the standings
		fmt.Println("UserName,Points")
		for _, standing := range standings {
			fmt.Printf("%s,%d\n", standing.UserName, standing.Points)
		}

		return
	}

	if reportFlag != nil && *reportFlag != "" {
		// Parse the rally ID from the command line argument
		rallyId := points.ParseStringToUint(*reportFlag)

		// Get points for the rally
		scored, err := getRallyPoints(rallyId, store, config)
		if err != nil {
			log.Fatalf("Failed to get rally points: %v", err)
		}

		// Fetch championship points
		standings, err := points.FetchChampionshipPoints(store, config)
		if err != nil {
			log.Fatalf("Failed to fetch championship points: %v", err)
		}

		// Prepare report data
		reportData := reports.ReportData{
			Rally:        scored,
			Championship: standings,
		}

		// Export the report to markdown file
		if err := reports.ExportMarkdown("report.md", reportData); err != nil {
			log.Fatalf("Failed to export report: %v", err)
		}

		fmt.Println("Report exported to report.md")
		return
	}
}

func createRally(rallyId uint64, config *configuration.Config, store *database.Store) error {
	// Set the rally in the database
	if err := points.StoreRally(rallyId, config, store); err != nil {
		return fmt.Errorf("Failed to store rally: %w", err)
	}

	err := points.StoreOverall(rallyId, store, config)
	if err != nil {
		return fmt.Errorf("Failed to store overall rally data: %w", err)
	}

	err = points.StoreStages(rallyId, store, config)
	if err != nil {
		return fmt.Errorf("Failed to store rally stage data: %w", err)
	}

	return nil
}

func getRallyPoints(rallyId uint64, store *database.Store, config *configuration.Config) ([]points.ScoreRecord, error) {
	// Fetch the overall results from the database
	overallData, err := points.FetchRallyOverallDB(rallyId, store)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch overall data: %w", err)
	}

	// Assign points to the overall results
	scored, err := points.AssignPointsOverall(overallData, config)
	if err != nil {
		return nil, fmt.Errorf("Failed to assign points: %w", err)
	}

	return scored, nil
}
