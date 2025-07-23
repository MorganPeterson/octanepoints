package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/database"
	"git.sr.ht/~nullevoid/octanepoints/grab"
	"git.sr.ht/~nullevoid/octanepoints/reports"
)

const configPath string = "config.toml" // Path to the configuration file

func main() {
	// flags
	createRallyFlag := flag.String("create", "", "create rally with given ID number")
	reportFlag := flag.String("report", "", "export report to markdown file")
	summaryFlag := flag.Bool("summary", false, "fetch driver summaries")
	driverReportFlag := flag.String("driver", "", "export driver report to markdown file")
	grabFlag := flag.String("grab", "", "grab raw rally data with given ID number")
	classReportFlag := flag.String("class", "", "export class report to markdown file")

	flag.Parse()

	if grabFlag != nil && *grabFlag != "" {
		// Grab raw rally data
		if err := grab.Grab(context.Background(), *grabFlag); err != nil {
			log.Fatalf("Failed to grab rally data: %v", err)
		}
		log.Printf("Rally %s setup successfully.\n", *grabFlag)
		return
	}

	// Load the configuration
	config := configuration.MustLoad(configPath)

	// init database store
	store, err := database.NewStore(config.Database.Name)
	if err != nil {
		log.Fatalf("Failed to initialize database store: %v", err)
	}
	defer store.Close()

	if createRallyFlag != nil && *createRallyFlag != "" {
		// Create a new rally in the database
		if err := database.CreateRally(*createRallyFlag, config, store); err != nil {
			log.Fatalf("Failed to create rally: %v", err)
		}
		log.Printf("Rally %s created successfully.\n", *createRallyFlag)
		return
	}

	if reportFlag != nil && *reportFlag != "" {
		// Export the report to markdown file
		if err := reports.ExportReport(*reportFlag, store, config); err != nil {
			log.Fatalf("Failed to export report: %v", err)
		}

		fmt.Println("Report exported to report.md")
		return
	}

	if summaryFlag != nil && *summaryFlag {
		if err := reports.ExportDriverSummaries(store, config); err != nil {
			log.Fatalf("Failed to export driver summaries: %v", err)
		}
		fmt.Println("Driver summaries exported to driver_summaries.md")
		return
	}

	if driverReportFlag != nil && *driverReportFlag != "" {
		if err := reports.DriverRallyReport(*driverReportFlag, store, config); err != nil {
			log.Fatalf("Failed to export driver summary: %v", err)
		}
		fmt.Println("Driver summary exported to driver_summary.md")
		return
	}

	if classReportFlag != nil && *classReportFlag != "" {
		if err := reports.ExportClassReport(*classReportFlag, store, config); err != nil {
			log.Fatalf("Failed to export class report: %v", err)
		}
		fmt.Println("Class report exported to class_summaries.md")
		return
	}
}
