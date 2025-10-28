package configuration

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// RallyDescription represents the structure of a rally description TOML file.
type RallyDescription struct {
	Rally Rally `toml:"rally"`
}

// Rally mirrors the [rally] table in the TOML.
type Rally struct {
	RallyId          int64   `toml:"rallyId"  json:"rallyId"`
	Name             string  `toml:"name"     json:"name"`
	Description      string  `toml:"description" json:"description"`
	Creator          string  `toml:"creator"  json:"creator"`
	DamageLevel      string  `toml:"damageLevel" json:"damageLevel"`
	NumberOfLegs     int64   `toml:"numberOfLegs" json:"numberOfLegs"`
	SuperRally       bool    `toml:"superRally" json:"superRally"`
	PacenotesOptions string  `toml:"pacenotesOptions" json:"pacenotesOptions"`
	Started          int64   `toml:"started"  json:"started"`  // Unix seconds
	Finished         int64   `toml:"finished" json:"finished"` // Unix seconds
	TotalDistance    float64 `toml:"totalDistance" json:"totalDistance"`
	CarGroups        string  `toml:"carGroups" json:"carGroups"` // comma-separated
	StartAt          string  `toml:"startAt"  json:"startAt"`    // e.g. RFC3339 or free text
	EndAt            string  `toml:"endAt"    json:"endAt"`
}

// CarGroupList returns a normalized list of car groups (split/trim) without
// changing the TOML schema.
func (r Rally) CarGroupList() []string {
	if strings.TrimSpace(r.CarGroups) == "" {
		return nil
	}
	parts := strings.Split(r.CarGroups, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// Validate performs basic semantic checks and light normalization.
func (d *RallyDescription) Validate() error {
	r := d.Rally

	if strings.TrimSpace(r.Name) == "" {
		return errors.New("rally.name must be set")
	}
	if r.NumberOfLegs <= 0 {
		return fmt.Errorf("rally.numberOfLegs must be > 0 (got %d)", r.NumberOfLegs)
	}
	if r.TotalDistance < 0 {
		return fmt.Errorf("rally.totalDistance must be >= 0 (got %f)", r.TotalDistance)
	}

	if r.DamageLevel == "" {
		return fmt.Errorf("damageLevel must be set, got: %s", r.DamageLevel)
	}

	// Started/Finished unix seconds consistency (if both provided).
	if r.Started > 0 && r.Finished > 0 && r.Finished < r.Started {
		return fmt.Errorf("rally.finished (%d) < rally.started (%d)", r.Finished, r.Started)
	}
	// Optional: parse StartAt/EndAt if they look like RFC3339 and check ordering.
	const layout = time.RFC3339
	if t1, err1 := time.Parse(layout, r.StartAt); err1 == nil {
		if t2, err2 := time.Parse(layout, r.EndAt); err2 == nil && t2.Before(t1) {
			return fmt.Errorf("rally.endAt (%s) is before rally.startAt (%s)", r.EndAt, r.StartAt)
		}
	}
	return nil
}

// DecodeRally decodes a TOML rally description from an io.Reader,
// fails on unknown keys, and validates the result.
func DecodeRally(r io.Reader) (*RallyDescription, error) {
	var desc RallyDescription
	md, err := toml.NewDecoder(r).Decode(&desc)
	if err != nil {
		return nil, fmt.Errorf("parsing rally file: %w", err)
	}
	if undec := md.Undecoded(); len(undec) > 0 {
		return nil, fmt.Errorf("unknown config key(s): %v", undec)
	}
	if err := desc.Validate(); err != nil {
		return nil, err
	}
	return &desc, nil
}

// LoadRally reads the TOML file at path and decodes it via DecodeRally.
func LoadRally(path string) (*RallyDescription, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("rally file not found: %w", err)
	}
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open rally file: %w", err)
	}
	defer f.Close()
	return DecodeRally(f)
}

// MustLoadRally is a convenience for CLI-style programs.
func MustLoadRally(path string) *RallyDescription {
	d, err := LoadRally(path)
	if err != nil {
		panic(err)
	}
	return d
}
