// db/database.go
package database

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"git.sr.ht/~nullevoid/octanepoints/parser"
)

// CreateRally initializes a rally in the database by setting its description,
// overall results, and stages based on the provided rally ID and configuration.
func CreateRally(rallyId int64, config *configuration.Config, store *Store) error {
	if err := setRally(rallyId, store, config); err != nil {
		return fmt.Errorf("Failed to store rally: %w", err)
	}

	err := setOverall(rallyId, store, config)
	if err != nil {
		return fmt.Errorf("Failed to store overall rally data: %w", err)
	}

	err = setStages(rallyId, store, config)
	if err != nil {
		return fmt.Errorf("Failed to store rally stage data: %w", err)
	}

	return nil
}

// GetDriversRallySummary fetches the summary of drivers for a specific rally that
// did not DNF (Did Not Finish) from the database table rally_overalls.
func GetDriversRallySummary(store *Store, opts *QueryOpts) ([]RallyOverall, error) {
	var recs []RallyOverall
	err := store.DB.Order("time3 asc").
		Where("rally_id = ? AND time3 > 0", *opts.RallyId).
		Find(&recs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch driver rally summary: %w", err)
	}

	return recs, nil
}

// GetDriverStages fetches the stages for drivers in a rally from the database.
func GetDriverStages(store *Store, rallyId int64, userName string) ([]StageSummary, error) {
	var stages []StageSummary
	err := store.DB.Raw(DriverStagesQuery(), rallyId, userName).Find(&stages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stages summary for %s: %w", userName, err)
	}
	return stages, nil
}

// GetRallyOverall fetches the overall results for a rally from the database table
// rally_overalls. If the results are not found, it returns an error.
func GetRallyOverall(store *Store, opts *QueryOpts) ([]RallyOverall, error) {
	// Fetch all overall records from the database
	var recs []RallyOverall

	if opts != nil {
		err := store.DB.Order("time3 asc").Where("rally_id = ?", *opts.RallyId).Find(&recs).Error
		if err != nil {
			return nil, fmt.Errorf("fetching overall records: %w", err)
		}
	} else {
		err := store.DB.Order("time3 asc").Find(&recs).Error
		if err != nil {
			return nil, fmt.Errorf("fetching overall records: %w", err)
		}
	}

	return recs, nil
}

// GetRankedRows reads out the class points rows from the database for a
// given rally ID and for the championship as a whole.
func GetRankedRows(store *Store, opts *QueryOpts) ([]RankedRow, error) {
	var rows []RankedRow

	sql, err := FetchedRowsQuery(opts)
	if err != nil {
		return nil, err
	}

	var args []any
	if opts.RallyId != nil {
		args = append(args, *opts.RallyId)
	}

	if err := store.DB.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetSeasonSummaryQuery returns the SQL query to fetch the season summary.
func GetSeasonSummary(store *Store, config *configuration.Config) ([]DriverSummary, error) {
	var sums []DriverSummary
	sql := GetSeasonSummaryQuery(config)
	if err := store.DB.Raw(sql).Scan(&sums).Error; err != nil {
		return sums, err
	}

	sort.Slice(sums, func(i, j int) bool {
		return sums[i].TotalChampionshipPoints > sums[j].TotalChampionshipPoints
	})

	return sums, nil
}

// GetRallyUserNames fetches the unique user names of drivers who participated
// in a specific rally from the database.
func GetRallyUserNames(store *Store, rallyId int64) ([]string, error) {
	var userNames []string
	err := store.DB.Model(&RallyStage{}).
		Where("rally_id = ?", rallyId).
		Distinct("user_name").
		Pluck("user_name", &userNames).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user names: %w", err)
	}

	return userNames, nil
}

// GetClasses fetches all classes from the database and returns them as a map
// with the class ID as the key.
func GetClasses(store *Store) (map[int64]Class, error) {
	var cs []Class
	if err := store.DB.Model(&Class{}).Scan(&cs).Error; err != nil {
		return nil, err
	}
	m := make(map[int64]Class, len(cs))
	for _, c := range cs {
		m[c.ID] = c
	}
	return m, nil
}

// fetchCsv reads a CSV file from the specified path and returns its content as
// a slice of string slices. It assumes the CSV uses semicolons as delimiters.
func fetchCsv(path string, config *configuration.Config) ([][]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}
	filePath := filepath.Join(
		currentDir,
		config.General.Directory,
		config.Download.Directory,
		path)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening CSV file %s: %w", filePath, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	if len(config.Download.Delimiter) != 1 {
		return nil, fmt.Errorf("delimiter is not set in the configuration")
	}
	reader.Comma = rune(config.Download.Delimiter[0]) // Use the first character as the delimiter

	return reader.ReadAll()
}

// SetOverall stores the overall results from the CSV file into the database.
func setOverall(rallyId int64, store *Store, config *configuration.Config) error {
	csvPath := filepath.Join(
		fmt.Sprintf("%d", rallyId),
		fmt.Sprintf("%d_%s", rallyId, config.Download.OverallFileName),
	)

	r, err := fetchCsv(csvPath, config)
	if err != nil {
		return err
	}

	var allCars []Cars
	if err := store.DB.Find(&allCars).Error; err != nil {
		return fmt.Errorf("failed to preload all cars: %w", err)
	}
	carMap := make(map[string]Cars, len(allCars))
	for _, car := range allCars {
		carMap[car.Slug] = car
	}

	var recs []RallyOverall
	for _, row := range r[1:] { // skip header row
		// CSV columns: #;userid;user_name;real_name;nationality;car;time3;super_rally;penalty
		// split car into name and brand based off of the first space in the string
		carSlug := parser.Slugify(row[5])
		car, ok := carMap[carSlug]
		if !ok {
			return fmt.Errorf("car %s not found in database", carSlug)
		}
		err = store.DB.Where("slug = ?", carSlug).Find(&car).Error
		if err != nil {
			return fmt.Errorf("failed to find car %s: %w", carSlug, err)
		}

		rec := RallyOverall{
			RallyId:     rallyId,
			UserId:      parser.StringToInt(row[1]),
			Position:    row[0],
			UserName:    row[2],
			RealName:    row[3],
			Nationality: row[4],
			Car:         row[5],
			CarID:       car.ID,
			Time3:       parser.HMS(row[6]),
			SuperRally:  parser.StringToInt(row[7]),
			Penalty:     parser.StringToFloat(row[8]),
		}

		recs = append(recs, rec)
	}

	if err := store.DB.Create(&recs).Error; err != nil {
		return fmt.Errorf("batch insert rally overall records in database: %w", err)
	}

	return nil
}

// setRally stores the rally information in the database.
func setRally(rallyId int64, store *Store, config *configuration.Config) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	rallyPath := filepath.Join(
		currentDir,
		config.General.Directory,
		config.Download.Directory,
		fmt.Sprintf("%d", rallyId),
		fmt.Sprintf("%d.toml", rallyId),
	)
	desc, err := configuration.LoadRally(rallyPath)
	if err != nil {
		return fmt.Errorf("loading rally description: %w", err)
	}

	// Convert the loaded description into a Rally struct
	rally := &Rally{
		RallyId:          desc.Rally.RallyId,
		Name:             desc.Rally.Name,
		Description:      desc.Rally.Description,
		Creator:          desc.Rally.Creator,
		DamageLevel:      desc.Rally.DamageLevel,
		NumberOfLegs:     desc.Rally.NumberOfLegs,
		SuperRally:       desc.Rally.SuperRally,
		PacenotesOptions: desc.Rally.PacenotesOptions,
		Started:          desc.Rally.Started,
		Finished:         desc.Rally.Finished,
		TotalDistance:    desc.Rally.TotalDistance,
		CarGroups:        desc.Rally.CarGroups,
	}
	if desc.Rally.StartAt != "" {
		startAt, err := time.Parse("2006-01-02 15:04", desc.Rally.StartAt)
		if err != nil {
			return fmt.Errorf("parsing start time: %w", err)
		}
		rally.StartAt = startAt
	}
	if desc.Rally.EndAt != "" {
		endAt, err := time.Parse("2006-01-02 15:04", desc.Rally.EndAt)
		if err != nil {
			return fmt.Errorf("parsing end time: %w", err)
		}
		rally.EndAt = endAt
	}

	// Store the rally information in the database
	if err := store.DB.Create(rally).Error; err != nil {
		return fmt.Errorf("storing rally in database: %w", err)
	}

	return nil
}

// setStages stores the stages from the CSV file into the database.
func setStages(rallyId int64, store *Store, config *configuration.Config) error {
	csvPath := filepath.Join(
		fmt.Sprintf("%d", rallyId),
		fmt.Sprintf("%d_%s", rallyId, config.Download.StageFileName),
	)
	r, err := fetchCsv(csvPath, config)
	if err != nil {
		return err
	}

	var recs []RallyStage
	for _, row := range r[1:] { // skip header row
		if len(row) < 16 {
			return fmt.Errorf("malformed row (len=%d): %v", len(row), row)
		}
		rec := RallyStage{
			RallyId:        rallyId,
			StageNum:       parser.StringToInt(row[0]),
			StageName:      row[1],
			Nationality:    row[2],
			UserName:       row[3],
			RealName:       row[4],
			Group:          row[5],
			CarName:        row[6],
			Time1:          parser.StringToFloat(row[7]),
			Time2:          parser.StringToFloat(row[8]),
			Time3:          parser.StringToFloat(row[9]),
			FinishRealTime: parser.RealTime(row[10]),
			Penalty:        parser.StringToFloat(row[11]),
			ServicePenalty: parser.StringToFloat(row[12]),
			SuperRally:     parser.StringToBool(row[13]),
			Progress:       row[14],
			Comments:       row[15],
		}

		recs = append(recs, rec)
	}

	if err := store.DB.Create(&recs).Error; err != nil {
		return fmt.Errorf("batch insert rally stage records in database: %w", err)
	}

	return nil
}
