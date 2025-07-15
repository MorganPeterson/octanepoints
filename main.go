package main

import (
	"flag"
	"fmt"
	"log"
	"sort"

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
	summaryFlag := flag.Bool("summary", false, "fetch driver summaries")

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

	if summaryFlag != nil && *summaryFlag {
		// Fetch driver summaries
		summaries, err := getDriverSummaries(store)
		if err != nil {
			log.Fatalf("Failed to fetch driver summaries: %v", err)
		}

		if err := reports.ExportDriverSummaries("driver_summaries.md", summaries); err != nil {
			log.Fatalf("Failed to export driver summaries: %v", err)
		}
		fmt.Println("Driver summaries exported to driver_summaries.md")
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

func getDriverSummaries(store *database.Store) ([]database.DriverSummary, error) {
	var sums []database.DriverSummary

	sql := `
    SELECT
      ro.user_name,
      ro.nationality,
      COUNT(DISTINCT ro.rally_id) AS rallies_started,
      SUM(CASE CAST(ro.position AS INTEGER) WHEN 1 THEN 1 ELSE 0 END) AS rally_wins,
      SUM(CASE WHEN CAST(ro.position AS INTEGER) <= 3 THEN 1 ELSE 0 END)  AS podiums,
      MIN(CAST(ro.position AS INTEGER))                 AS best_position,
      AVG(CAST(ro.position AS INTEGER))                 AS average_position,

      -- total super‑rallied stages
      (SELECT COUNT(*)
         FROM rally_stages rs
        WHERE rs.user_name = ro.user_name
          AND rs.super_rally = 1
      )                                                 AS total_super_rallied_stages,

      -- total stage wins
      (SELECT COUNT(*)
         FROM (
           SELECT rs2.user_name
             FROM rally_stages rs2
             JOIN (
               SELECT rally_id, stage_num, MIN(time3) AS min_time
                 FROM rally_stages
                GROUP BY rally_id, stage_num
             ) AS sw
               ON rs2.rally_id = sw.rally_id
              AND rs2.stage_num = sw.stage_num
              AND rs2.time3 = sw.min_time
         ) AS winners
        WHERE winners.user_name = ro.user_name
      )                                                 AS stage_wins,

      -- championship points per rally (adjust values as you prefer)
      (SELECT SUM(
         CASE CAST(ro2.position AS INTEGER)
           WHEN 1 THEN 32
           WHEN 2 THEN 28
           WHEN 3 THEN 25
           WHEN 4 THEN 22
           WHEN 5 THEN 20
           WHEN 6 THEN 18
           WHEN 7 THEN 16
           WHEN 8 THEN 14
           WHEN 9 THEN 12
           WHEN 10 THEN 11
		   WHEN 11 THEN 10
		   WHEN 12 THEN 9
		   WHEN 13 THEN 8
		   WHEN 14 THEN 7
		   WHEN 15 THEN 6
		   WHEN 16 THEN 5
		   WHEN 17 THEN 4
		   WHEN 18 THEN 3
		   WHEN 19 THEN 2
		   WHEN 20 THEN 1
           ELSE 0
         END
       )
       FROM rally_overalls ro2
      WHERE ro2.user_name = ro.user_name
      )                                                 AS total_championship_points

    FROM rally_overalls ro
    GROUP BY ro.user_name, ro.nationality
    ORDER BY ro.user_name;
    `

	if err := store.DB.Raw(sql).Scan(&sums).Error; err != nil {
		return nil, err
	}

	sort.Slice(sums, func(i, j int) bool {
		return sums[i].TotalChampionshipPoints > sums[j].TotalChampionshipPoints
	})

	return sums, nil
}
