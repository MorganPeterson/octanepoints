package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
	"git.sr.ht/~nullevoid/octanepoints/grab"
	"git.sr.ht/~nullevoid/octanepoints/reports"
)

const configPath string = "config.toml" // Path to the configuration file

func main() {
	// flags
	var createRally int64     // too insert all rally data into the database
	var basicReport int64     // get basic points data for a single rally and overall
	var rallySummary bool     // get a summary of all points and other stats so far
	var driverSummaries int64 // get a all driver summaries for a single rally
	var grabData int64        // grab data for a rally and download it from rsf
	var classReport int64     // get class points data for a single rally and overall

	flag.Int64Var(&createRally, "create", 0, "put rally data in db with given ID number")
	flag.Int64Var(&basicReport, "report", 0, "export rally points report for a single rally to markdown file")
	flag.BoolVar(&rallySummary, "summary", false, "fetch driver point summaries for the championship so far")
	flag.Int64Var(&driverSummaries, "driver", 0, "export driver report for a single rally to markdown file")
	flag.Int64Var(&grabData, "grab", 0, "grab raw rally data from RSF with given rally ID number")
	flag.Int64Var(&classReport, "class", 0, "export class points report for a single rally to markdown file")

	flag.Parse()

	// Load the configuration
	config := configuration.MustLoad(configPath)

	err := ensureDir(config.General.Directory)
	if err != nil {
		log.Fatalf("Failed to ensure general directory exists: %v", err)
	}

	downloadDir := filepath.Join(config.General.Directory, config.Download.Directory)
	reportDir := filepath.Join(config.General.Directory, config.Report.Directory)
	databaseDir := filepath.Join(config.General.Directory, config.Database.Directory)

	err = ensureDir(downloadDir)
	if err != nil {
		log.Fatalf("Failed to ensure download directory exists: %v", err)
	}

	err = ensureDir(reportDir)
	if err != nil {
		log.Fatalf("Failed to ensure description directory exists: %v", err)
	}

	err = ensureDir(databaseDir)
	if err != nil {
		log.Fatalf("Failed to ensure database directory exists: %v", err)
	}

	// get a single rally's data from the RSF rally page and download it
	// This does not require the database to be set up or configuration to be
	// loaded and so it is at the top of the main function.
	if grabData != 0 {
		if err := grab.Grab(context.Background(), grabData, config); err != nil {
			log.Fatalf("Failed to grab rally data: %v", err)
		}
		log.Printf("Rally %d setup successfully.\n", grabData)
		return
	}

	// get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	dbFilePath := filepath.Join(
		currentDir,
		config.General.Directory,
		config.Database.Directory,
		config.Database.Name,
	)
	// init database store
	store, err := database.NewStore(dbFilePath)
	if err != nil {
		log.Fatalf("Failed to initialize database store: %v", err)
	}
	defer store.Close()

	// Given a rally ID number, we read the raw csv data for a rally from the
	// rallies/[rally_id] directory and put it into the database.
	if createRally != 0 {
		if err := database.CreateRally(createRally, config, store); err != nil {
			log.Fatalf("Failed to create rally: %v", err)
		}
		log.Printf("Rally %d created successfully.\n", createRally)
		return
	}

	// This will read data from the database given a single rally ID number. It
	// will then assign points to drivers and create a table for the single rally.
	// It will also produce a table with total points awarded to drivers across
	// all rallies in the championship.
	if basicReport != 0 {
		// Export the report to markdown file
		if err := reports.ExportReport(basicReport, store, config); err != nil {
			log.Fatalf("Failed to export report: %v", err)
		}

		fmt.Println("Report exported to report.md")
		return
	}

	// Rally summary is a table with more detailed statistics about the
	// championship so far, including driver points, averages, stage wins, etc.
	if rallySummary {
		if err := reports.ExportDriverSummaries(store, config); err != nil {
			log.Fatalf("Failed to export driver summaries: %v", err)
		}
		fmt.Println("Driver summaries exported to driver_summaries.md")
		return
	}

	// Driver summaries is 2 tables. The first table is a small amount of stats
	// compared to averages of the single rally. The second is a stage-by-stage
	// summary of the rally for each driver.
	if driverSummaries != 0 {
		if err := reports.DriverRallyReport(driverSummaries, store, config); err != nil {
			log.Fatalf("Failed to export driver summary: %v", err)
		}
		fmt.Println("Driver summary exported to driver_summary.md")
		return
	}

	// Class report is like the basic report, but it groups drivers by class
	// and shows the points awarded to each class in the championship and that rally.
	if classReport != 0 {
		if err := reports.ExportClassReport(classReport, store, config); err != nil {
			log.Fatalf("Failed to export class report: %v", err)
		}
		fmt.Println("Class report exported to class_summaries.md")
		return
	}
}

func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}
