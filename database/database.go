// db/database.go
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DriverSummary holds all of your 10 summary metrics.
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

// #;userid;user_name;real_name;nationality;car;time3;super_rally;penalty
type RallyOverall struct {
	ID          uint64        `gorm:"primaryKey;autoIncrement"` // Add an ID field for GORM
	RallyId     uint64        `gorm:"not null"`                 // Use uint64 for RallyId
	UserId      uint64        `gorm:"not null"`                 // Use uint64 for UserId
	Position    string        `gorm:"size:255;not null"`
	UserName    string        `gorm:"size:255;not null"`
	RealName    string        `gorm:"size:255;not null"`
	Nationality string        `gorm:"size:255;not null"`
	Car         string        `gorm:"size:255;not null"`
	Time3       time.Duration `gorm:"not null"`
	SuperRally  int64         `gorm:"not null"`
	Penalty     float64       `gorm:"default:0"` // Use float64 for penalty, default to 0
}

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
	return s.DB.AutoMigrate(
		&RallyOverall{},
		&RallyStage{},
		&Rally{}, // add additional models here
	)
}

// Close cleanly shuts down the database connection.
func (s *Store) Close() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
