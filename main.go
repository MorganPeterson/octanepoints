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
	var allReports int64      // runs all above commands in one go

	flag.Int64Var(&createRally, "create", 0, "put rally data in db with given ID number")
	flag.Int64Var(&basicReport, "report", 0, "export rally points report for a single rally to markdown file")
	flag.BoolVar(&rallySummary, "summary", false, "fetch driver point summaries for the championship so far")
	flag.Int64Var(&driverSummaries, "driver", 0, "export driver report for a single rally to markdown file")
	flag.Int64Var(&grabData, "grab", 0, "grab raw rally data from RSF with given rally ID number")
	flag.Int64Var(&classReport, "class", 0, "export class points report for a single rally to markdown file")
	flag.Int64Var(&allReports, "all", 0, "run all commands for a single rally in one go")

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
		log.Fatalf("Failed to ensure report base directory exists: %v", err)
	}

	err = ensureDir(filepath.Join(reportDir, config.Report.MdDirectory))
	if err != nil {
		log.Fatalf("Failed to ensure report markdown directory exists: %v", err)
	}

	err = ensureDir(filepath.Join(reportDir, config.Report.CsvDirectory))
	if err != nil {
		log.Fatalf("Failed to ensure report csv directory exists: %v", err)
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

	// If all reports are requested we do all of the above commands in one go.
	if allReports != 0 {
		// If all reports are requested, we run the grab command first
		if err := grab.Grab(context.Background(), allReports, config); err != nil {
			log.Fatalf("Failed to grab rally data for all reports: %v", err)
		}
		log.Printf("Rally %d setup successfully.\n", allReports)

		if err := database.CreateRally(allReports, config, store); err != nil {
			log.Fatalf("Failed to create rally: %v", err)
		}
		log.Printf("Rally %d created successfully.\n", allReports)

		if err := reports.ExportReport(allReports, store, config); err != nil {
			log.Fatalf("Failed to export %d_points_summary_report: %v", allReports, err)
		}
		log.Printf("Report exported to %d_points_summary_report\n", allReports)

		if err := reports.ExportDriverSummaries(store, config); err != nil {
			log.Fatalf("Failed to export drivers_summary: %v", err)
		}
		log.Println("Championship summary exported to drivers_summary")

		if err := reports.DriverRallyReport(allReports, store, config); err != nil {
			log.Fatalf("Failed to export %d_driver_rally_summary: %v", allReports, err)
		}
		log.Printf("Driver rally summary exported to %d_driver_rally_summary\n", allReports)

		if err := reports.ExportClassReport(allReports, store, config); err != nil {
			log.Fatalf("Failed to export %d_class_summary: %v", allReports, err)
		}
		log.Printf("Class report exported to %d_class_summary\n", allReports)

		return
	}

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
			log.Fatalf("Failed to export %d_points_summary_report: %v", basicReport, err)
		}

		fmt.Printf("Report exported to %d_points_summary_report\n", basicReport)
		return
	}

	// Rally summary is a table with more detailed statistics about the
	// championship so far, including driver points, averages, stage wins, etc.
	if rallySummary {
		if err := reports.ExportDriverSummaries(store, config); err != nil {
			log.Fatalf("Failed to export drivers_summary: %v", err)
		}
		fmt.Println("Championship summary exported to drivers_summary")
		return
	}

	// Driver summaries is 2 tables. The first table is a small amount of stats
	// compared to averages of the single rally. The second is a stage-by-stage
	// summary of the rally for each driver.
	if driverSummaries != 0 {
		if err := reports.DriverRallyReport(driverSummaries, store, config); err != nil {
			log.Fatalf("Failed to export %d_driver_rally_summary: %v", driverSummaries, err)
		}
		fmt.Printf("Driver rally summary exported to %d_driver_rally_summary\n", driverSummaries)
		return
	}

	// Class report is like the basic report, but it groups drivers by class
	// and shows the points awarded to each class in the championship and that rally.
	if classReport != 0 {
		if err := reports.ExportClassReport(classReport, store, config); err != nil {
			log.Fatalf("Failed to export %d_class_summary: %v", classReport, err)
		}
		fmt.Printf("Class report exported to %d_class_summary\n", classReport)
		return
	}
}

func ensureDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}
