package parser

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

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

func StringToBool(s string) bool {
	var value bool
	switch s {
	case "1":
		value = true
	default:
		value = false
	}
	return value
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
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
