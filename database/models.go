package database

import "time"

// Cars represents a car in the database in the table cars.
type Cars struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`                  // Add an ID field for GORM
	RSFID    int64  `gorm:"not null;uniqueIndex" json:"RSFID"`         // Use uint64 for the RSF car ID
	Slug     string `gorm:"size:255;uniqueIndex;not null" json:"Slug"` // Slug for the car
	Brand    string `gorm:"size:255;not null" json:"Brand"`            // Name of the car
	Model    string `gorm:"size:255;not null" json:"Model"`            // Model of the car
	Category string `gorm:"size:255;not null" json:"Category"`         // Category of the car
}

// Class represents a class in the database in the table classes.
type Class struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`      // Add an ID field for GORM
	Name        string `gorm:"size:255;uniqueIndex;not null"` // Name of the class
	Slug        string `gorm:"size:255;uniqueIndex;not null"` // Slug for the class
	Description string `gorm:"size:512"`                      // Description of the class
	Active      bool   `gorm:"not null;default:true"`         // Whether the class is active
}

// ClassCar represents the many-to-many relationship between classes and cars.
type ClassCar struct {
	ClassID int64 `gorm:"primaryKey;index:idx_cc_class_id"` // Class ID
	CarID   int64 `gorm:"primaryKey;index:idx_cc_car_id"`   // Car ID
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

type QueryOpts struct {
	RallyId int64 // Optional rally ID to filter by
}

// Rally represents a rally overview in the database.
type Rally struct {
	ID               int64     `gorm:"primaryKey;autoIncrement"` // Add an ID field for GORM
	RallyId          int64     `gorm:"not null;uniqueIndex"`     // Use uint64 for RallyId
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

// RallyOverall represents the overall results of a rally for a driver.
type RallyOverall struct {
	ID          int64         `gorm:"primaryKey;autoIncrement"`       // Add an ID field for GORM
	RallyId     int64         `gorm:"not null;index:idx_ro_rally_id"` // Use uint64 for RallyId
	UserId      int64         `gorm:"not null"`                       // Use uint64 for UserId
	Position    string        `gorm:"size:255;not null"`
	UserName    string        `gorm:"size:255;not null"`
	RealName    string        `gorm:"size:255;not null"`
	Nationality string        `gorm:"size:255;not null"`
	Car         string        `gorm:"size:255;not null"`
	CarID       int64         `gorm:"not null;index:idx_ro_car_id"` // CarID
	Time3       time.Duration `gorm:"not null"`
	SuperRally  int64         `gorm:"not null"`
	Penalty     float64       `gorm:"default:0"` // Use float64 for penalty, default to 0
}

// RallyStage represents a stage in a rally.
type RallyStage struct {
	ID             int64     `gorm:"primaryKey;autoIncrement"` // Add an ID field for GORM
	RallyId        int64     `gorm:"not null"`
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
	RallyId  int64
	ClassId  int64
	UserId   int64
	UserName string
	Time3    int64
	Pos      int64
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
