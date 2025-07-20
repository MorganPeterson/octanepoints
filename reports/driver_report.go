package reports

import (
	"fmt"

	"git.sr.ht/~nullevoid/octanepoints/database"
)

var (
	summarySql = `
WITH 
  -- 1) only real finishers (time3>0), compute total_time per stage
  stage_totals AS (
    SELECT
      stage_num,
      stage_name,
      user_name,
      time3 
        + penalty 
        + service_penalty  AS total_time,
      penalty + service_penalty AS penalty,
      comments
    FROM rally_stages
    WHERE rally_id   = :rally_id
      AND time3      >  0                -- <<< filter out DNF’s
  ),

  -- 2) find the winning total_time per stage (among finishers)
  min_totals AS (
    SELECT
      stage_num,
      MIN(total_time) AS winner_time
    FROM stage_totals
    GROUP BY stage_num
  ),

  -- 3) rank every finisher on each stage
  ranked AS (
    SELECT
      st.*,
      ROW_NUMBER() OVER (
        PARTITION BY stage_num
        ORDER BY total_time ASC
      ) AS position
    FROM stage_totals st
  )

-- 4) pull out just your driver, joining back the winner_time
SELECT
  r.stage_num,
  r.stage_name,
  r.position,
  r.total_time       AS stage_time,
  r.total_time - mt.winner_time  AS delta_to_winner,
  r.penalty,
  r.comments
FROM ranked r
JOIN min_totals mt  USING (stage_num)
WHERE r.user_name = :user_name
ORDER BY r.stage_num;
`

	driverOverallSql = `
`
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

func StagesSummary(
	rallyId uint64, store *database.Store,
) (map[string][]StageSummary, error) {
	var userNames []string
	err := store.DB.Model(&database.RallyStage{}).
		Where("rally_id = ?", rallyId).
		Distinct("user_name").
		Pluck("user_name", &userNames).Error
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user names: %w", err)
	}

	var summary = make(map[string][]StageSummary)
	tables, _ := store.DB.Migrator().GetTables()
	fmt.Println("Tables in the database:", tables)
	for _, userName := range userNames {
		var stages []StageSummary
		err = store.DB.Raw(summarySql, rallyId, userName).Scan(&stages).Error
		if err != nil {
			return nil, fmt.Errorf("failed to fetch stages summary for %s: %w", userName, err)
		}
		summary[userName] = stages
	}

	return summary, nil
}
