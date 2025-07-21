package reports

import (
	"fmt"
	"log"
	"sort"
	"time"

	"git.sr.ht/~nullevoid/octanepoints/database"
)

type StageSummary struct {
	StageNum      int64   `json:"stage_num"`
	StageName     string  `json:"stage_name"`
	Position      int64   `json:"position"`
	StageTime     float64 `json:"stage_time"`
	DeltaToWinner float64 `json:"delta_to_winner"`
	Penalty       float64 `json:"penalty"`
	Comments      string  `json:"comments"`
}

type SummaryRow struct {
	Metric      string
	DriverValue string
	FieldAvg    string
	RankText    string
}

type DriverReport struct {
	Stages  []StageSummary
	Overall []SummaryRow
}

func getDriverOverallSummary(rallyId uint64, userName string, store *database.Store) ([]SummaryRow, error) {
	var overall []database.RallyOverall
	err := store.DB.Where("rally_id = ?", rallyId).
		Find(&overall).Error
	if err != nil {
		log.Printf("no overall summary for %s: %v", userName, err)
	}

	var finishers []database.RallyOverall
	for _, r := range overall {
		if r.Time3 > 0 {
			finishers = append(finishers, r)
		}
	}

	if len(finishers) == 0 {
		return nil, fmt.Errorf("no finishers found for rally %d", rallyId)
	}

	sort.Slice(finishers, func(i, j int) bool {
		return finishers[i].Time3 < finishers[j].Time3
	})

	totalDrivers := len(finishers)
	winnerTime := finishers[0].Time3

	var sumTime time.Duration
	var sumPenalty float64
	var sumSuper int64
	for _, r := range overall {
		sumTime += r.Time3
		sumPenalty += r.Penalty
		sumSuper += r.SuperRally
	}

	avgTime := time.Duration(int64(sumTime) / int64(totalDrivers))
	avgPenalty := sumPenalty / float64(totalDrivers)
	avgSuper := float64(sumSuper) / float64(totalDrivers)

	var driver *database.RallyOverall
	var finishRank int
	for i, r := range finishers {
		if r.UserName == userName {
			driver = &r
			finishRank = i + 1
			break
		}
	}

	if driver == nil {
		// they have dnf'd; look at full list
		for i, r := range overall {
			if r.UserName == userName {
				driver = &r
				finishRank = i + 1
				break
			}
		}
	}
	rows := []SummaryRow{
		{
			Metric: "Finishing Position",
			DriverValue: func() string {
				if driver == nil || driver.Time3 == 0 {
					return "DNF"
				}
				return fmt.Sprintf("%d", finishRank)
			}(),
			FieldAvg: "-",
			RankText: func() string {
				if driver == nil || driver.Time3 == 0 {
					return ""
				}
				return fmt.Sprintf("%d/%d", finishRank, totalDrivers)
			}(),
		},
		{
			Metric: "Total Time",
			DriverValue: func() string {
				if driver == nil || driver.Time3 == 0 {
					return "DNF"
				}
				return formatDuration(driver.Time3)
			}(),
			FieldAvg: formatDuration(avgTime),
			RankText: "",
		},
		{
			Metric: "Delta to Winner",
			DriverValue: func() string {
				if driver == nil || driver.Time3 == 0 {
					return "DNF"
				}
				delta := driver.Time3 - winnerTime
				return formatDuration(delta)
			}(),
			FieldAvg: "-",
			RankText: "",
		},
		{
			Metric: "Total Penalty",
			DriverValue: func() string {
				if driver == nil {
					return "DNF"
				}
				return fmt.Sprintf("%.0fs", driver.Penalty)
			}(),
			FieldAvg: fmt.Sprintf("%.1fs", avgPenalty),
			RankText: "",
		},
		{
			Metric: "Super Rallies",
			DriverValue: func() string {
				if driver == nil {
					return "DNF"
				}
				return fmt.Sprintf("%d", driver.SuperRally)
			}(),

			FieldAvg: fmt.Sprintf("%.1f", avgSuper),
			RankText: "",
		},
	}

	return rows, nil
}

// formatDuration prints d as HH:MM:SS.ss (hours, minutes, seconds, centiseconds)
func formatDuration(d time.Duration) string {
	// total hundredths of a second
	totalHundredths := int(d.Nanoseconds() / 1e7) // 1e9 ns/sec ÷ 100 = 1e7
	hundredths := totalHundredths % 100

	totalSeconds := totalHundredths / 100
	seconds := totalSeconds % 60

	totalMinutes := totalSeconds / 60
	minutes := totalMinutes % 60

	hours := totalMinutes / 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d.%02d", hours, minutes, seconds, hundredths)
	}

	if minutes > 0 {
		return fmt.Sprintf("%d:%02d.%02d", minutes, seconds, hundredths)
	}

	return fmt.Sprintf("%d.%02d", seconds, hundredths)
}
