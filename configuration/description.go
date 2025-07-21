package configuration

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config is the top‐level representation of your TOML file.
// Add or remove fields / nested structs as your application requires.
type Description struct {
	Rally struct {
		RallyId          uint64  `toml:"rallyId"`          // Unique identifier for the rally
		Name             string  `toml:"name"`             // Name of the rally
		Description      string  `toml:"description"`      // Description of the rally
		Creator          string  `toml:"creator"`          // Creator of the rally
		DamageLevel      string  `toml:"damageLevel"`      // Damage level of the rally
		NumberOfLegs     int64   `toml:"numberOfLegs"`     // Number of legs in the rally
		SuperRally       bool    `toml:"superRally"`       // Whether the rally is a super rally
		PacenotesOptions string  `toml:"pacenotesOptions"` // Pacenotes options for the rally
		Started          int64   `toml:"started"`          // Start time of the rally in Unix timestamp
		Finished         int64   `toml:"finished"`         // Finish time of the rally in Unix timestamp
		TotalDistance    float64 `toml:"totalDistance"`    // Total distance of the rally in kilometers
		CarGroups        string  `toml:"carGroups"`        // Car groups allowed in the rally
		StartAt          string  `toml:"startAt"`          // Start time of the rally
		EndAt            string  `toml:"endAt"`            // End time of the rally
	} `toml:"rally"` // Nested struct for rally configuration
}

// LoadRally reads the TOML file at path, decodes into Description, and
// applies any sensible defaults. It returns an error if parsing fails
func LoadRally(path string) (*Description, error) {
	desc, err := loadDescription(path)
	if err != nil {
		return nil, fmt.Errorf("loading rally description: %w", err)
	}

	return desc, nil
}

// LoadDescription reads the TOML file at path, decodes into Description, and
// applies any sensible defaults. It returns an error if parsing fails
func loadDescription(path string) (*Description, error) {
	// Make sure file exists
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("rally file not found: %w", err)
	}

	var desc Description
	if _, err := toml.DecodeFile(path, &desc); err != nil {
		return nil, fmt.Errorf("parsing rally file: %w", err)
	}

	return &desc, nil
}
