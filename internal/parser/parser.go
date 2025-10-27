package parser

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

func RealTime(val string) time.Time {
	FinishRealTime, err := time.Parse("2006-01-02 15:04:05", val)
	if err != nil {
		log.Printf("Error parsing FinishRealTime: %v", err)
		return time.Time{}
	}
	return FinishRealTime
}

// FmtDuration formats a time.Duration into a string in "MM:SS.sss" or
// "HH:MM:SS.sss" format.
func FmtDuration(d time.Duration) string {
	neg := d < 0
	if neg {
		d = -d
	}
	h := int(d / time.Hour)
	d -= time.Duration(h) * time.Hour
	m := int(d / time.Minute)
	d -= time.Duration(m) * time.Minute
	s := float64(d) / float64(time.Second)

	if h > 0 {
		if neg {
			return fmt.Sprintf("-%d:%02d:%05.2f", h, m, s)
		}
		return fmt.Sprintf("%d:%02d:%05.2f", h, m, s)
	}
	if neg {
		return fmt.Sprintf("-%d:%05.2f", m, s)
	}
	return fmt.Sprintf("%d:%05.2f", m, s)
}

// HMS parses a string in "MM:SS.sss" or "HH:MM:SS.sss" format into a time.Duration.
// It returns an error if the format is invalid.
func HMS(s string) time.Duration {
	var h, m int
	var sec float64

	var duration time.Duration
	var err error

	switch strings.Count(s, ":") {
	case 1:
		// MM:SS.sss
		_, err = fmt.Sscanf(s, "%d:%f", &m, &sec)
	case 2:
		// HH:MM:SS.sss
		_, err = fmt.Sscanf(s, "%d:%d:%f", &h, &m, &sec)
	default:
		log.Printf("invalid time format %q", s)
		return 0
	}

	if err != nil {
		log.Printf("invalid HMS value %q: %v", s, err)
		return 0
	}

	duration += time.Duration(h) * time.Hour
	duration += time.Duration(m) * time.Minute
	duration += time.Duration(sec * float64(time.Second))

	return duration
}

func StringToBool(s string) bool {
	return s == "1"
}

func StringToFloat(s string) float64 {
	if s == "" {
		return 0
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return value
}

func StringToInt(s string) int64 {
	if s == "" {
		return 0
	}
	value, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		fmt.Printf("Error parsing uint64: %v\n", err)
		return 0
	}
	return value
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	// replace all non-alphanum with '-'
	s = slugRegex.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
