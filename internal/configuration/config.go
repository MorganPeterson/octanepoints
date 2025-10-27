package configuration

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"gorm.io/gorm"
)

const (
	defaultReportDir   = "season_reports" // Default directory for reports
	defaultDataDir     = "data"           // Default directory for data
	defaultDatabaseDir = "database"       // Default directory for database files
	defaultDownloadDir = "rallies"        // Default directory for downloaded rally data
	defaultDelimiter   = ";"              // Default CSV delimiter
)

var defaultPoints = []int64{
	32, 28, 25, 22, 20, 18, 16, 14, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1,
}

// Config is just for TOML decoding; you typically won’t insert
// this table directly.
type Config struct {
	General  General  `toml:"general"`
	Download Download `toml:"download"`
	Report   Report   `toml:"report"`
	Database Database `toml:"database"`
	Classes  []Class  `toml:"classes"`
}

type ConfigMeta struct {
	gorm.Model
	FilePath string `gorm:"uniqueIndex;not null"` // Path to the TOML file
	Checksum string `gorm:"not null"`             // Checksum of the TOML file
}

// General maps the [general] section.
// Points and ClassPoints are stored as JSON in SQLite. :contentReference[oaicite:8]{index=8}
type General struct {
	gorm.Model
	Points      []int64 `toml:"points"       gorm:"serializer:json"` // [32,28,…]
	ClassPoints []int64 `toml:"classPoints"  gorm:"serializer:json"`
	ClassesType string  `toml:"classesType"` // "driver"
	Directory   string  `toml:"directory"`   // "data"
}

// Download maps the [download] section. :contentReference[oaicite:9]{index=9}
type Download struct {
	gorm.Model
	RallyCSVURLTmpl     string `toml:"rallyCSVURLTmpl"`     // e.g. "https://…?rally_id=%d"
	RallyCSVOverallTmpl string `toml:"rallyCSVOverallTmpl"` // e.g. "https://…?rally_id=%d&cg=7"
	Directory           string `toml:"directory"`           // "rallies"
	StageFileName       string `toml:"stageFileName"`       // "table.csv"
	OverallFileName     string `toml:"overallFileName"`     // "All_table.csv"
	Delimiter           string `toml:"delimiter"`           // ";"
}

// Report maps the [report] section, embedding its subtables.
type Report struct {
	Directory    string        `toml:"directory"`    // "rally_reports"
	Format       string        `toml:"format"`       // "markdown" or "csv" or "both"
	MdDirectory  string        `toml:"mdDirectory"`  // "markdown"
	CsvDirectory string        `toml:"csvDirectory"` // "csv"
	Delimiter    string        `toml:"delimiter"`    // ";"
	Class        ReportClass   `toml:"class"`
	Points       ReportPoints  `toml:"points"`
	Drivers      ReportDrivers `toml:"drivers"`
}

type ReportClass struct {
	SummaryFilename string `toml:"summaryFilename"` // "class_summary"
}

type ReportPoints struct {
	SummaryFileName string `toml:"summaryFileName"` // "points_summary"
}

type ReportDrivers struct {
	SeasonSummaryFilename string `toml:"seasonSummaryFilename"` // "drivers_summary"
	RallySummaryFilename  string `toml:"rallySummaryFilename"`  // "drivers_rally_summary"
}

// Database maps the [database] section. :contentReference[oaicite:14]{index=14}
type Database struct {
	gorm.Model
	Name      string `toml:"name"`      // "season1.db"
	Directory string `toml:"directory"` // "database"
}

// Class maps each [[classes]] entry.
// name, description, categories, drivers :contentReference[oaicite:15]{index=15}
// (categories and drivers stored as JSON arrays)
type Class struct {
	gorm.Model
	Name        string   `toml:"name"`        // e.g. "Gold"
	Description string   `toml:"description"` // e.g. "Gold Class Drivers"
	Categories  []string `toml:"categories"`
	Drivers     []string `toml:"drivers"`
}

// validate sets defaults and enforces required fields.
func (c *Config) validate() error {
	if len(c.General.Points) == 0 {
		c.General.Points = defaultPoints // Use default points if none specified
	}

	if len(c.General.ClassPoints) == 0 {
		c.General.ClassPoints = defaultPoints // Use default class points if none specified
	}
	if c.Report.Directory == "" {
		c.Report.Directory = defaultReportDir // Use default report directory if none specified
	}

	if c.Report.Format == "" {
		c.Report.Format = "markdown" // Use markdown as default format
	}
	// Validate format is one of the supported options
	if c.Report.Format != "markdown" && c.Report.Format != "csv" && c.Report.Format != "both" {
		return fmt.Errorf("invalid report format '%s': must be 'markdown', 'csv', or 'both'", c.Report.Format)
	}

	if c.Report.MdDirectory == "" {
		c.Report.MdDirectory = "markdown" // Use markdown as default directory
	}

	if c.Report.CsvDirectory == "" {
		c.Report.CsvDirectory = "csv" // Use csv as default directory
	}

	if c.Report.Delimiter == "" {
		c.Report.Delimiter = defaultDelimiter // Use default delimiter if none specified
	}

	if c.General.Directory == "" {
		c.General.Directory = defaultDataDir // Use default data directory if none specified
	}

	if c.Download.Directory == "" {
		c.Download.Directory = defaultDownloadDir // Use general directory as default for download
	}

	if c.Database.Directory == "" {
		c.Database.Directory = defaultDatabaseDir // Use general directory as default for database
	}
	return nil
}

// Load reads the TOML file at path, decodes into Config, and
// applies any sensible defaults. It returns an error if parsing fails
// or if required fields are missing.
func Load(path string) (*Config, error) {
	// Make sure file exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("config file not found: %w", err)
	}

	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

// MustLoad is like Load but panics on error. Useful in init().
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %+v", err))
	}

	// Apply defaults and validation
	if err := cfg.validate(); err != nil {
		panic(fmt.Sprintf("failed to load config: %+v", err))
	}

	return cfg
}
