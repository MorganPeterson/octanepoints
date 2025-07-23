package reports

import (
	"fmt"
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
	Stages  []database.StageSummary
	Overall []SummaryRow
}

type DriverReportConfig struct {
	Overall   []database.RallyOverall
	Finishers []database.RallyOverall
	Config    RallyConfig
}

type RallyConfig struct {
	WinnerTime   time.Duration
	TotalDrivers int
	AvgTime      time.Duration
	AvgPenalty   float64
	AvgSuper     float64
}

func configSummaries(rallyId uint64, store *database.Store) (DriverReportConfig, error) {
	overall, err := database.GetRallyOverall(store, &database.QueryOpts{RallyId: rallyId})
	if err != nil {
		return DriverReportConfig{}, fmt.Errorf("no overall summary error=%v", err)
	}

	finishers, err := database.GetDriversRallySummary(store, &database.QueryOpts{RallyId: rallyId})
	if err != nil {
		return DriverReportConfig{}, fmt.Errorf("no finishers error=%v", err)
	}

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

	return DriverReportConfig{
		Overall:   overall,
		Finishers: finishers,
		Config: RallyConfig{
			WinnerTime:   winnerTime,
			TotalDrivers: totalDrivers,
			AvgTime:      avgTime,
			AvgPenalty:   avgPenalty,
			AvgSuper:     avgSuper,
		},
	}, nil
}

func getDriverOverallSummary(name string, config DriverReportConfig) ([]SummaryRow, error) {
	var driver *database.RallyOverall
	var finishRank int
	for i, r := range config.Finishers {
		if r.UserName == name {
			driver = &r
			finishRank = i + 1
			break
		}
	}

	if driver == nil {
		// they have dnf'd; look at full list
		for i, r := range config.Overall {
			if r.UserName == name {
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
				return fmt.Sprintf("%d/%d", finishRank, config.Config.TotalDrivers)
			}(),
		},
		{
			Metric: "Total Time",
			DriverValue: func() string {
				if driver == nil || driver.Time3 == 0 {
					return "DNF"
				}
				return fmtDur(driver.Time3)
			}(),
			FieldAvg: fmtDur(config.Config.AvgTime),
			RankText: "",
		},
		{
			Metric: "Delta to Winner",
			DriverValue: func() string {
				if driver == nil || driver.Time3 == 0 {
					return "DNF"
				}
				delta := driver.Time3 - config.Config.WinnerTime
				return fmtDur(delta)
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
			FieldAvg: fmt.Sprintf("%.1fs", config.Config.AvgPenalty),
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

			FieldAvg: fmt.Sprintf("%.1f", config.Config.AvgSuper),
			RankText: "",
		},
	}

	return rows, nil
}
