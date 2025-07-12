package configuration

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

const (
	defaultCG  int    = 7 // Default CG value
	defaultURL string = "https://rallysimfans.hu/rbr/csv_export_results.php?rally_id=%d&cg=%d"
)

var defaultPoints = []int{
	32, 28, 25, 22, 20, 18, 16, 14, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1,
}

// Config is the top‐level representation of your TOML file.
// Add or remove fields / nested structs as your application requires.
type Config struct {
	CG      int    `toml:"cg"`      // e.g. 7
	Headers string `toml:"headers"` // e.g. "pos,driver,points"
	Points  []int  `toml:"points"`  // e.g. [32, 28, 25, ...]
	URL     string `toml:"url"`     // e.g. "https://example.com/data.csv"
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

	// Apply defaults and validation
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// MustLoad is like Load but panics on error. Useful in init().
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// validate sets defaults and enforces required fields.
func (c *Config) validate() error {
	if c.CG == 0 {
		c.CG = defaultCG // Set default CG if not specified
	}

	if c.URL == "" {
		c.URL = defaultURL // Set default URL if not specified
	}

	if len(c.Points) == 0 {
		c.Points = defaultPoints // Use default points if none specified
	}

	return nil
}
