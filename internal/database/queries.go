package database

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/MorganPeterson/octanepoints/internal/configuration"
	"github.com/goccy/go-json"
)

var _ embed.FS

//go:embed sql_files/get_season_summary.sql
var getSeasonSummarySQL string

// GetSeasonSummaryQuery returns the SQL query to fetch the season summary.
func GetSeasonSummary(store *Store, config *configuration.Config) ([]DriverSummary, error) {
	var sums []DriverSummary

	pnts, err := json.Marshal(config.General.Points)
	if err != nil {
		return sums, err
	}

	if err := store.DB.Raw(CleanSQL(getSeasonSummarySQL), string(pnts)).Scan(&sums).Error; err != nil {
		return sums, err
	}

	sort.Slice(sums, func(i, j int) bool {
		return sums[i].TotalChampionshipPoints > sums[j].TotalChampionshipPoints
	})

	return sums, nil
}

//go:embed sql_files/get_driver_stages.sql
var getDriverStagesSQL string

// GetDriverStages fetches the stages for drivers in a rally from the database.
func GetDriverStages(store *Store, rallyId int64, userName string) ([]StageSummary, error) {
	var stages []StageSummary
	err := store.DB.Raw(CleanSQL(getDriverStagesSQL), rallyId, userName).Find(&stages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stages summary for %s: %w", userName, err)
	}
	return stages, nil
}

// GetRankedRows reads out the class points rows from the database for a
// given rally ID and for the championship as a whole.
func GetRankedRows(store *Store, opts *QueryOpts) ([]RankedRow, error) {
	var rows []RankedRow
	var err error

	if opts.Type != nil {
		if *opts.Type == DRIVER_CLASS {
			rows, err = getDriverClassRankings(store, opts)
		} else {
			rows, err = getCarClassRankings(store, opts)
		}
	} else {
		return rows, fmt.Errorf("query type is not set")
	}

	return rows, err
}

func CleanSQL(query string) string {
	q := strings.TrimSpace(query)
	q = strings.TrimRight(q, " \t\r\n;")
	return q
}

//go:embed sql_files/get_car_class_rankings.sql
var getCarClassRankingsSQL string

func getCarClassRankings(store *Store, opts *QueryOpts) ([]RankedRow, error) {
	var params any
	if opts.RallyId != nil {
		params = int64(*opts.RallyId)
	} else {
		params = sql.NullInt64{}
	}

	var rankings []RankedRow
	if err := store.DB.Raw(CleanSQL(getCarClassRankingsSQL), params).Scan(&rankings).Error; err != nil {
		return nil, err
	}

	return rankings, nil
}

//go:embed sql_files/get_driver_class_rankings.sql
var getDriverClassRankingsSQL string

func getDriverClassRankings(store *Store, opts *QueryOpts) ([]RankedRow, error) {
	var rallId *int64 = nil
	if opts.RallyId != nil {
		rallId = opts.RallyId
	} else {
		rallId = nil
	}

	var rankings []RankedRow
	if err := store.DB.Raw(CleanSQL(getDriverClassRankingsSQL), rallId).Scan(&rankings).Error; err != nil {
		return nil, err
	}

	return rankings, nil
}
