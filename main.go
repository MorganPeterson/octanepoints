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

var (
	createRally     = flag.Int64("create", 0, "put rally data in db with given ID number")
	basicReport     = flag.Int64("report", 0, "export rally points report for a single rally to markdown file")
	rallySummary    = flag.Bool("summary", false, "fetch driver point summaries for the championship so far")
	driverSummaries = flag.Int64("driver", 0, "export driver report for a single rally to markdown file")
	grabData        = flag.Int64("grab", 0, "grab raw rally data from RSF with given rally ID number")
	classReport     = flag.Int64("class", 0, "export class points report for a single rally to markdown file")
	allReports      = flag.Int64("all", 0, "run all commands for a single rally in one go")
)

func main() {
	flag.Parse()

	var active []string
	flag.Visit(func(f *flag.Flag) {
		active = append(active, f.Name)
	})

	if len(active) == 0 || len(active) > 1 {
		log.Fatalf("You must specify exactly one command to run. Active flags: %v", active)
	}

	// Load the configuration
	config := configuration.MustLoad(configPath)

	if err := createDirs(config); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	// get a single rally's data from the RSF rally page and download it
	// This does not require the database to be set up or configuration to be
	// loaded and so it is at the top of the main function.
	if active[0] == "grab" && grabData != nil {
		if err := grab.Grab(context.Background(), *grabData, config); err != nil {
			log.Fatalf("Failed to grab rally data: %v", err)
		}
		log.Printf("Rally %d setup successfully.\n", *grabData)
		return
	}

	dbFilePath := filepath.Join(
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

	switch active[0] {
	case "all":
		doAllReports(store, config, allReports)
	case "create":
		doCreateRally(store, config, createRally)
	case "report":
		doReport(store, config, basicReport)
	case "summary":
		doSummary(store, config)
	case "driver":
		doDriver(store, config, driverSummaries)
	case "class":
		doClass(store, config, classReport)
	default:
		log.Fatalf("Unknown command: %s. Active flags: %v", active[0], active)
	}
}

func ensureDirs(dirs []string) error {
	if len(dirs) == 0 {
		return nil
	}

	// create the root directory
	root := dirs[0]
	if err := os.MkdirAll(root, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", root, err)
	}

	// create each subsequent as a subdirectory of root
	for _, sub := range dirs[1:] {
		path := filepath.Join(root, sub)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create sub-directory %q: %w", path, err)
		}
	}
	return nil
}

func createDirs(config *configuration.Config) error {
	baseDirs := []string{
		config.General.Directory,
		config.Download.Directory,
		config.Database.Directory,
		filepath.Join(config.Report.Directory, config.Download.Directory),
		filepath.Join(config.Report.Directory, config.Report.Directory),
	}

	err := ensureDirs(baseDirs)
	if err != nil {
		return fmt.Errorf("Failed to ensure general directory exists: %v", err)
	}

	return nil
}

// doAllReports Given a rally ID number, we run all reports in one go.
// This is useful for generating all reports for a single rally in one command.
func doAllReports(store *database.Store, config *configuration.Config, rallyId *int64) {
	var rid int64
	if rallyId == nil {
		log.Fatal("Rally ID must be provided for all reports")
	}

	rid = *rallyId

	// If all reports are requested, we run the grab command first
	if err := grab.Grab(context.Background(), rid, config); err != nil {
		log.Fatalf("Failed to grab rally data for all reports: %v", err)
	}
	log.Printf("Rally %d downloaded successfully.\n", allReports)

	if err := database.CreateRally(rid, config, store); err != nil {
		log.Fatalf("Failed to create rally: %v", err)
	}
	log.Printf("Rally %d created successfully.\n", allReports)

	if err := reports.ExportReport(rid, store, config); err != nil {
		log.Fatalf("Failed to export %d_points_summary_report: %v", allReports, err)
	}
	log.Printf("Report exported to %d_points_summary_report\n", allReports)

	if err := reports.ExportDriverSummaries(store, config); err != nil {
		log.Fatalf("Failed to export drivers_summary: %v", err)
	}
	log.Println("Championship summary exported to drivers_summary")

	if err := reports.DriverRallyReport(rid, store, config); err != nil {
		log.Fatalf("Failed to export %d_driver_rally_summary: %v", allReports, err)
	}
	log.Printf("Driver rally summary exported to %d_driver_rally_summary\n", allReports)

	if err := reports.ExportClassReport(rid, store, config); err != nil {
		log.Fatalf("Failed to export %d_class_summary: %v", allReports, err)
	}
	log.Printf("Class report exported to %d_class_summary\n", allReports)
}

// doCreateRally Given a rally ID number, we read the raw csv data for a rally from the
// rallies/[rally_id] directory and put it into the database.
func doCreateRally(store *database.Store, config *configuration.Config, rallyId *int64) {
	if rallyId == nil {
		log.Fatal("Rally ID must be provided for creating a rally")
	}

	if err := database.CreateRally(*rallyId, config, store); err != nil {
		log.Fatalf("Failed to create rally: %v", err)
	}
	log.Printf("Rally %d created successfully.\n", *rallyId)
}

// doReport will read data from the database given a single rally ID number. It
// will then assign points to drivers and create a table for the single rally.
// It will also produce a table with total points awarded to drivers across
// all rallies in the championship.
func doReport(store *database.Store, config *configuration.Config, rallyId *int64) {
	if basicReport == nil {
		log.Fatal("Rally ID must be provided for basic report")
	}

	// Export the report to markdown file
	if err := reports.ExportReport(*rallyId, store, config); err != nil {
		log.Fatalf("Failed to export %d_points_summary_report: %v", *rallyId, err)
	}

	fmt.Printf("Report exported to %d_points_summary_report\n", *rallyId)
}

// doSummary will export the championship summary and driver summaries.
func doSummary(store *database.Store, config *configuration.Config) {
	if err := reports.ExportDriverSummaries(store, config); err != nil {
		log.Fatalf("Failed to export drivers_summary: %v", err)
	}
	fmt.Println("Championship summary exported to drivers_summary")
}

// doDriver will export the driver rally summary for a single rally.
// Driver summaries is 2 tables. The first table is a small amount of stats
// compared to averages of the single rally. The second is a stage-by-stage
// summary of the rally for each driver.
func doDriver(store *database.Store, config *configuration.Config, rallyId *int64) {
	if rallyId == nil {
		log.Fatal("Rally ID must be provided for driver summaries")
	}
	if err := reports.DriverRallyReport(*rallyId, store, config); err != nil {
		log.Fatalf("Failed to export %d_driver_rally_summary: %v", *rallyId, err)
	}
	fmt.Printf("Driver rally summary exported to %d_driver_rally_summary\n", *rallyId)
}

// doClass will export the class report for a single rally.
// The class report groups drivers by class and shows the points awarded to
// each class in the championship and that rally.
func doClass(store *database.Store, config *configuration.Config, classReport *int64) {
	if classReport == nil {
		log.Fatal("Rally ID must be provided for class report")
	}
	if err := reports.ExportClassReport(*classReport, store, config); err != nil {
		log.Fatalf("Failed to export %d_class_summary: %v", *classReport, err)
	}
	fmt.Printf("Class report exported to %d_class_summary\n", *classReport)
}
