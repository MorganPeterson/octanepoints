// db/database.go
package database

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Class struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`      // Add an ID field for GORM
	Name        string `gorm:"size:255;uniqueIndex;not null"` // Name of the class
	Slug        string `gorm:"size:255;uniqueIndex;not null"` // Slug for the class
	Description string `gorm:"size:512"`                      // Description of the class
	Active      bool   `gorm:"not null;default:true"`         // Whether the class is active
}

type ClassCar struct {
	ClassID uint64 `gorm:"primaryKey;index:idx_cc_class_id"` // Class ID
	CarID   uint64 `gorm:"primaryKey;index:idx_cc_car_id"`   // Car ID
}

// Cars represents a car in the database in the table cars.
type Cars struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`                  // Add an ID field for GORM
	RSFID    uint64 `gorm:"not null;uniqueIndex" json:"RSFID"`         // Use uint64 for the RSF car ID
	Slug     string `gorm:"size:255;uniqueIndex;not null" json:"Slug"` // Slug for the car
	Brand    string `gorm:"size:255;not null" json:"Brand"`            // Name of the car
	Model    string `gorm:"size:255;not null" json:"Model"`            // Model of the car
	Category string `gorm:"size:255;not null" json:"Category"`         // Category of the car
}

type ClassDriver struct {
	UserID  uint64 `gorm:"not null;uniqueIndex"`           // Use uint64 for UserID
	ClassID uint64 `gorm:"not null;index:idx_cd_class_id"` // Class ID
}

type Driver struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement"`             // Add an ID field for GORM
	UserID      uint64 `gorm:"not null;uniqueIndex"`                 // Use uint64 for UserID
	UserName    string `gorm:"size:255;not null" json:"UserName"`    // Name of the driver
	RealName    string `gorm:"size:255;not null" json:"RealName"`    // Real name of the driver
	Nationality string `gorm:"size:255;not null" json:"Nationality"` // Nationality of the driver
}

// DriverSummary holds all of the 10 summary metrics.
type DriverSummary struct {
	UserName                string  `gorm:"column:user_name"`
	Nationality             string  `gorm:"column:nationality"`
	RalliesStarted          int64   `gorm:"column:rallies_started"`
	RallyWins               int64   `gorm:"column:rally_wins"`
	Podiums                 int64   `gorm:"column:podiums"`
	StageWins               int64   `gorm:"column:stage_wins"`
	BestPosition            int64   `gorm:"column:best_position"`
	AveragePosition         float64 `gorm:"column:average_position"`
	TotalSuperRalliedStages int64   `gorm:"column:total_super_rallied_stages"`
	TotalChampionshipPoints int64   `gorm:"column:total_championship_points"`
}

// Rally represents a rally in the database. It is parsed from a TOML file
// and stored in the database in the table rallies.
type Rally struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement"` // Add an ID field for GORM
	RallyId          uint64    `gorm:"not null;uniqueIndex"`     // Use uint64 for RallyId
	Name             string    `gorm:"size:255;not null"`        // Name of the rally
	Description      string    `gorm:"not null"`                 // Description of the rally
	Creator          string    `gorm:"size:255;not null"`        // Creator of the rally
	DamageLevel      string    `gorm:"size:255;not null"`        // Damage level of the rally
	NumberOfLegs     int64     `gorm:"not null"`                 // Number of legs in the rally
	SuperRally       bool      `gorm:"not null"`                 // Whether the rally is a super rally
	PacenotesOptions string    `gorm:"size:255;not null"`        // Pacenotes options for the rally
	Started          int64     `gorm:"not null"`                 // Start time of the rally in Unix timestamp
	Finished         int64     `gorm:"not null"`                 // Finish time of the rally in Unix timestamp
	TotalDistance    float64   `gorm:"not null"`                 // Total distance of the rally in kilometers
	CarGroups        string    `gorm:"not null"`                 // Car groups allowed in the rally
	StartAt          time.Time `gorm:"not null"`                 // Start time of the rally
	EndAt            time.Time `gorm:"not null"`                 // End time of the rally
}

// RallyOverall represents the overall results of a rally for a driver. It is
// stored in the database in the table rally_overalls.
type RallyOverall struct {
	ID          uint64        `gorm:"primaryKey;autoIncrement"`       // Add an ID field for GORM
	RallyId     uint64        `gorm:"not null;index:idx_ro_rally_id"` // Use uint64 for RallyId
	UserId      uint64        `gorm:"not null"`                       // Use uint64 for UserId
	Position    string        `gorm:"size:255;not null"`
	UserName    string        `gorm:"size:255;not null"`
	RealName    string        `gorm:"size:255;not null"`
	Nationality string        `gorm:"size:255;not null"`
	Car         string        `gorm:"size:255;not null"`
	CarID       uint64        `gorm:"not null;index:idx_ro_car_id"` // CarID
	Time3       time.Duration `gorm:"not null"`
	SuperRally  int64         `gorm:"not null"`
	Penalty     float64       `gorm:"default:0"` // Use float64 for penalty, default to 0
}

// RallyStage represents a stage in a rally. It is stored in the database in
// the table rally_stages.
type RallyStage struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"` // Add an ID field for GORM
	RallyId        uint64    `gorm:"not null"`
	StageNum       int64     `gorm:"not null"`          // Stage number in the rally
	StageName      string    `gorm:"size:255;not null"` // Name of the stage
	Nationality    string    `gorm:"size:255;not null"` // Drivers nationality
	UserName       string    `gorm:"size:255;not null"` // Username of the participant
	RealName       string    `gorm:"size:255;not null"` // Real name of the participant
	Group          string    `gorm:"size:255;not null"` // Group of the participant
	CarName        string    `gorm:"size:255;not null"` // Car name used in the stage
	Time1          float64   `gorm:"not null"`          // Time for the first run
	Time2          float64   `gorm:"not null"`          // Time for the second run
	Time3          float64   `gorm:"not null"`          // Time for the third run
	FinishRealTime time.Time `gorm:"not null"`          // Real finish time of the stage
	Penalty        float64   `gorm:"default:0"`         // Penalty time for the stage, default to 0
	ServicePenalty float64   `gorm:"default:0"`         // Service penalty time, default to 0
	SuperRally     bool      `gorm:"not null"`          // Whether the stage is part of a super rally
	Progress       string    `gorm:"not null"`          // Progress of the stage
	Comments       string    `gorm:"size:255;not null"` // Comments for the stage
}

type RankedRow struct {
	RallyId  uint64
	ClassId  uint64
	UserId   uint64
	UserName string
	Time3    int64
	Pos      int64
}

type QueryOpts struct {
	RallyId uint64 // Optional rally ID to filter by
}

// Store wraps your GORM DB instance.
type Store struct {
	DB *gorm.DB
}

// NewStore opens (or creates) the SQLite file at path, applies
// connection settings, and runs migrations.
func NewStore(path string) (*Store, error) {
	// Open with a bit of GORM logging enabled; adjust logger level if needed.
	gormDB, err := gorm.Open(
		sqlite.Open(path+"?_foreign_keys=on"),
		&gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open gorm sqlite: %w", err)
	}

	// Grab the underlying *sql.DB to set connection limits.
	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("getting sql.DB from gorm: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(time.Minute)

	store := &Store{DB: gormDB}
	if err := store.Migrate(); err != nil {
		return nil, fmt.Errorf("automigrate failed: %w", err)
	}

	return store, nil
}

// Migrate runs AutoMigrate on all your models.
func (s *Store) Migrate() error {
	if err := s.DB.AutoMigrate(
		&RallyOverall{},
		&RallyStage{},
		&Rally{},
		&Cars{},
		&Class{},
		&ClassCar{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	var count int64
	if err := s.DB.Model(&Cars{}).Count(&count).Error; err != nil {
		return fmt.Errorf("counting cars: %w", err)
	}
	if count == 0 {
		if err := seedFromJSON(s.DB, "cars.json"); err != nil {
			return fmt.Errorf("seeding cars: %w", err)
		}
	}

	return nil
}

// Close cleanly shuts down the database connection.
func (s *Store) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func CreateRally(rallyIdStr string, config *configuration.Config, store *Store) error {
	rallyId := ParseStringToUint(rallyIdStr)

	// Set the rally in the database
	if err := setRally(ParseStringToUint(rallyIdStr), store, config); err != nil {
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

// GetRallyOverall fetches the overall results for a rally from the database table
// rally_overalls. If the results are not found, it returns an error.
func GetRallyOverall(store *Store, opts *QueryOpts) ([]RallyOverall, error) {
	// Fetch all overall records from the database
	var recs []RallyOverall

	if opts != nil {
		err := store.DB.Order("time3 asc").Where("rally_id = ?", opts.RallyId).Find(&recs).Error
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

func GetDriversRallySummary(store *Store, opts *QueryOpts) ([]RallyOverall, error) {
	var recs []RallyOverall
	err := store.DB.Order("time3 asc").
		Where("rally_id = ? AND time3 > 0", opts.RallyId).
		Find(&recs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch driver rally summary: %w", err)
	}

	return recs, nil
}

type StageSummary struct {
	StageNum      int64   `json:"stage_num"`
	StageName     string  `json:"stage_name"`
	Position      int64   `json:"position"`
	StageTime     float64 `json:"stage_time"`
	DeltaToWinner float64 `json:"delta_to_winner"`
	Penalty       float64 `json:"penalty"`
	Comments      string  `json:"comments"`
}

func GetDriverStages(store *Store, rallyId uint64, userName string) ([]StageSummary, error) {
	var stages []StageSummary
	err := store.DB.Raw(DriverStagesQuery(), rallyId, userName).Scan(&stages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch stages summary for %s: %w", userName, err)
	}
	return stages, nil
}

func GetRallyUserNames(store *Store, rallyId uint64) ([]string, error) {
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

func GetClasses(store *Store) (map[uint64]Class, error) {
	var cs []Class
	if err := store.DB.Model(&Class{}).Scan(&cs).Error; err != nil {
		return nil, err
	}
	m := make(map[uint64]Class, len(cs))
	for _, c := range cs {
		m[c.ID] = c
	}
	return m, nil
}

func FetchRankedRows(store *Store, opts *QueryOpts) ([]RankedRow, error) {
	var rows []RankedRow

	base := `
WITH ranked AS (
  SELECT
    ro.rally_id,
    ro.user_id,
    ro.user_name,
    ro.time3,
    cc.class_id,
    ROW_NUMBER() OVER (
        PARTITION BY ro.rally_id, cc.class_id
        ORDER BY ro.time3
    ) AS pos
  FROM rally_overalls ro
  JOIN cars       c  ON c.id = ro.car_id
  JOIN class_cars cc ON cc.car_id = c.id
{{ if .RallyFilter }} WHERE ro.rally_id = ? {{ end }}
)
SELECT rally_id, class_id, user_id, user_name, time3, pos
FROM ranked
ORDER BY rally_id, class_id, pos;
`

	type qtpl struct {
		RallyFilter bool
	}

	var sql string
	t := template.Must(template.New("q").Parse(base))
	buf := &strings.Builder{}

	err := t.Execute(buf, qtpl{RallyFilter: opts != nil})
	if err != nil {
		return nil, err
	}

	sql = buf.String()

	var args []any
	if opts != nil {
		args = append(args, opts.RallyId)
	}

	if err := store.DB.Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SetOverall stores the overall results from the CSV file into the database.
func setOverall(rallyId uint64, store *Store, config *configuration.Config) error {
	csvPath := fmt.Sprintf("%d/%d_All_table.csv", rallyId, rallyId)
	r, err := fetchCsv(csvPath, config)
	if err != nil {
		return err
	}

	for _, row := range r[1:] { // skip header row
		// #;userid;user_name;real_name;nationality;car;time3;super_rally;penalty
		// split car into name and brand based off of the first space in the string
		var car Cars
		carSlug := slugify(row[5])
		err = store.DB.Where("slug = ?", carSlug).Find(&car).Error
		if err != nil {
			return fmt.Errorf("failed to find car %s: %w", carSlug, err)
		}

		rec := RallyOverall{
			RallyId:     rallyId,
			UserId:      ParseStringToUint(row[1]),
			Position:    row[0],
			UserName:    row[2],
			RealName:    row[3],
			Nationality: row[4],
			Car:         row[5],
			CarID:       car.ID,
			Time3:       parseHMS(row[6]),
			SuperRally:  parseStringToInt64(row[7]),
			Penalty:     parseStringToFloat(row[8]),
		}

		if err := store.DB.Create(&rec).Error; err != nil {
			return fmt.Errorf("storing record in database: %w", err)
		}
	}
	return nil
}

func setRally(rallyId uint64, store *Store, config *configuration.Config) error {
	rallyPath := fmt.Sprintf("%s/%d/%d.toml", config.General.DescriptionDir, rallyId, rallyId)
	desc, err := configuration.LoadRally(rallyPath)

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

	if err != nil {
		return fmt.Errorf("loading rally description: %w", err)
	}

	// Store the rally information in the database
	if err := store.DB.Create(rally).Error; err != nil {
		return fmt.Errorf("storing rally in database: %w", err)
	}

	return nil
}

// StoreStages stores the stages from the CSV file into the database.
func setStages(rallyId uint64, store *Store, config *configuration.Config) error {
	csvPath := fmt.Sprintf("%d/%d_table.csv", rallyId, rallyId)
	r, err := fetchCsv(csvPath, config)
	if err != nil {
		return err
	}

	for _, row := range r[1:] { // skip header row
		rec := RallyStage{
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
			return fmt.Errorf("storing record in database: %w", err)
		}
	}

	return nil
}

func fetchCsv(path string, config *configuration.Config) ([][]string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("getting current directory: %w", err)
	}
	filePath := fmt.Sprintf("%s/%s/%s", currentDir, config.General.DescriptionDir, path)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening CSV file %s: %w", filePath, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = ';'

	return reader.ReadAll()
}

func parseFinishRealTime(row []string) time.Time {
	FinishRealTime, err := time.Parse("2006-01-02 15:04:05", row[10])
	if err != nil {
		log.Printf("Error parsing FinishRealTime: %v", err)
		return time.Time{}
	}
	return FinishRealTime
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
			log.Printf("invalid minutes: %+v", err)
			return 0
		}
		secF, err = strconv.ParseFloat(parts[1], 64)
		if err != nil {
			log.Printf("invalid seconds: %+v", err)
			return 0
		}

	case 3:
		// HH:MM:SS.sss
		h, err = strconv.Atoi(parts[0])
		if err != nil {
			log.Printf("invalid hours: %v", err)
			return 0
		}
		m, err = strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("invalid minutes: %+v", err)
			return 0
		}
		secF, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			log.Printf("invalid seconds: %+v", err)
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

func ParseStringToUint(s string) uint64 {
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

func seedFromJSON(db *gorm.DB, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var wrapper struct {
		Cars []Cars `json:"cars"`
	}
	if err := json.Unmarshal(raw, &wrapper); err != nil {
		return err
	}
	if len(wrapper.Cars) == 0 {
		return nil
	}

	// slugify car names and ensure they are unique
	for i := range wrapper.Cars {
		wrapper.Cars[i].Slug = slugify(wrapper.Cars[i].Brand + " " + wrapper.Cars[i].Model)
		if wrapper.Cars[i].Slug == "" {
			return fmt.Errorf("car slug is empty for car: %s %s", wrapper.Cars[i].Brand, wrapper.Cars[i].Model)
		}
	}

	return db.Transaction(func(tx *gorm.DB) error {
		// 1) Insert cars (if empty)
		var carCount int64
		if err := tx.Model(&Cars{}).Count(&carCount).Error; err != nil {
			return err
		}
		if carCount == 0 {
			if err := tx.Create(&wrapper.Cars).Error; err != nil {
				return err
			}
		} else {
			// Ensure IDs are loaded if they already exist
			for i := range wrapper.Cars {
				var id uint64
				if err := tx.Model(&Cars{}).
					Select("id").
					Where("rsfid = ?", wrapper.Cars[i].RSFID).
					Scan(&id).Error; err != nil {
					return err
				}
				wrapper.Cars[i].ID = id
			}
		}

		// 2) Build distinct classes from car.Category
		type tmpClass struct {
			Name string
			Slug string
		}
		seen := map[string]struct{}{}
		classesToInsert := make([]Class, 0)
		for _, car := range wrapper.Cars {
			if _, ok := seen[car.Category]; ok {
				continue
			}
			seen[car.Category] = struct{}{}
			classesToInsert = append(classesToInsert, Class{
				Name:        car.Category,
				Slug:        slugify(car.Category),
				Description: "",
				Active:      true,
			})
		}

		// 3) Upsert classes (ignore if exists)
		if len(classesToInsert) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "slug"}},
				DoNothing: true,
			}).Create(&classesToInsert).Error; err != nil {
				return err
			}
		}

		// Get ids for all categories we saw
		var dbClasses []Class
		slugs := make([]string, 0, len(seen))
		for k := range seen {
			slugs = append(slugs, slugify(k))
		}
		if err := tx.Where("slug IN ?", slugs).Find(&dbClasses).Error; err != nil {
			return err
		}
		nameToID := make(map[string]uint64, len(dbClasses))
		for _, c := range dbClasses {
			nameToID[c.Name] = c.ID
		}

		// 4) Build ClassCar join rows
		ccRows := make([]ClassCar, 0, len(wrapper.Cars))
		for _, car := range wrapper.Cars {
			classID := nameToID[car.Category]
			ccRows = append(ccRows, ClassCar{
				ClassID: classID,
				CarID:   car.ID,
			})
		}

		// Upsert/ignore duplicates
		if len(ccRows) > 0 {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "class_id"}, {Name: "car_id"}},
				DoNothing: true,
			}).Create(&ccRows).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// replace all non-alphanum with '-'
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
