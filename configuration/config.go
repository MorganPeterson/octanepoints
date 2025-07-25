package configuration

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

var defaultPoints = []int64{
	32, 28, 25, 22, 20, 18, 16, 14, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1,
}

type ClassConfig struct {
	Name        string   `toml:"name"`        // Class name, e.g. "Pro"
	Slug        string   `toml:"slug"`        // Slug for the class
	Description string   `toml:"description"` // Description of the class
	Categories  []string `toml:"categories"`  // Categories for this class
	Drivers     []string `toml:"drivers"`     // Drivers in this class
}

type DatabaseConfig struct {
	Name string `toml:"name"` // Database name, e.g. "octanepoints.db"
}

type GeneralConfig struct {
	Points         []int64 `toml:"points"`         // Overall points for drivers
	ClassPoints    []int64 `toml:"classPoints"`    // Points for each class
	ClassesType    string  `toml:"classesType"`    // Type of classes, e.g. "car" or "driver"
	DescriptionDir string  `toml:"descriptionDir"` // Directory for rally descriptions, e.g. "rallies"
}

// Config is the top‐level representation of your TOML file.
// Add or remove fields / nested structs as your application requires.
type Config struct {
	Database DatabaseConfig `toml:"database"` // Nested struct for database configuration
	General  GeneralConfig  `toml:"general"`  // Nested struct for general configuration
	Classes  []ClassConfig  `toml:"classes"`  // Slice of classes with their own points
}

// validate sets defaults and enforces required fields.
func (c *Config) validate() error {
	if len(c.General.Points) == 0 {
		c.General.Points = defaultPoints // Use default points if none specified
	}

	if len(c.General.ClassPoints) == 0 {
		c.General.ClassPoints = defaultPoints // Use default class points if none specified
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
