package database

import (
	"fmt"
	"strings"
	"text/template"

	"git.sr.ht/~nullevoid/octanepoints/configuration"
)

func GetSeasonSummaryQuery(config *configuration.Config) string {
	part_one := `
    SELECT
      ro.user_name,
      ro.nationality,
      COUNT(DISTINCT ro.rally_id) AS rallies_started,
      SUM(CASE CAST(ro.position AS INTEGER) WHEN 1 THEN 1 ELSE 0 END) AS rally_wins,
      SUM(CASE WHEN CAST(ro.position AS INTEGER) <= 3 THEN 1 ELSE 0 END)  AS podiums,
      MIN(CAST(ro.position AS INTEGER))                 AS best_position,
      AVG(CAST(ro.position AS INTEGER))                 AS average_position,

      -- total super‑rallied stages
      (SELECT COUNT(*)
         FROM rally_stages rs
        WHERE rs.user_name = ro.user_name
          AND rs.super_rally = 1
      )                                                 AS total_super_rallied_stages,

      -- total stage wins
      (SELECT COUNT(*)
         FROM (
           SELECT rs2.user_name
             FROM rally_stages rs2
             JOIN (
               SELECT rally_id, stage_num, MIN(time3) AS min_time
                 FROM rally_stages
                GROUP BY rally_id, stage_num
             ) AS sw
               ON rs2.rally_id = sw.rally_id
              AND rs2.stage_num = sw.stage_num
              AND rs2.time3 = sw.min_time
         ) AS winners
        WHERE winners.user_name = ro.user_name
      )                                                 AS stage_wins,

      -- championship points per rally (adjust values as you prefer)
      (SELECT SUM(
         CASE CAST(ro2.position AS INTEGER)
	`

	for i, points := range config.General.Points {
		part_one += fmt.Sprintf("WHEN %d THEN %d ", i+1, points)
	}

	part_one += `
           ELSE 0
         END
       )
       FROM rally_overalls ro2
      WHERE ro2.user_name = ro.user_name
      )                                                 AS total_championship_points

    FROM rally_overalls ro
    GROUP BY ro.user_name, ro.nationality
    ORDER BY ro.user_name;
    `

	return part_one
}

func DriverStagesQuery() string {
	return `
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
}

func FetchedRowsQuery(opts *QueryOpts) (string, error) {
	var base string
	if opts.Type != nil && *opts.Type == DRIVER_CLASS {
		base = `
  WITH driver_classes AS (
  SELECT
    ro.rally_id,
    ro.user_id,
    ro.user_name,
    ro.time3,
    cd.class_id
  FROM rally_overalls ro
  JOIN class_drivers cd ON cd.user_id = ro.user_id
{{ if .RallyFilter }} WHERE ro.rally_id = ? {{ end }}
),
ranked AS (
  SELECT
    dc.rally_id, dc.class_id, dc.user_id, dc.user_name, dc.time3,
    ROW_NUMBER() OVER (PARTITION BY dc.rally_id, dc.class_id ORDER BY dc.time3) AS pos
  FROM driver_classes dc
)

SELECT
  r.rally_id,
  r.class_id,
  r.user_id,
  r.user_name,
  r.pos
FROM ranked r
ORDER BY r.rally_id, r.class_id, r.pos;
`
	} else {
		base = `
WITH ranked AS (
  SELECT
    ro.rally_id,
    ro.user_id,
    ro.user_name,
    ro.time3,
    cc.class_id,
    ROW_NUMBER() OVER (
        PARTITION BY ro.rally_id, cc.class_id
        ORDER BY ro.time3
    ) AS pos
  FROM rally_overalls ro
  JOIN cars       c  ON c.id = ro.car_id
  JOIN class_cars cc ON cc.car_id = c.id
{{ if .RallyFilter }} WHERE ro.rally_id = ? {{ end }}
)
SELECT rally_id, class_id, user_id, user_name, time3, pos
FROM ranked
ORDER BY rally_id, class_id, pos;
`
	}

	type qtpl struct {
		RallyFilter bool
	}

	t := template.Must(template.New("q").Parse(base))
	buf := &strings.Builder{}

	err := t.Execute(buf, qtpl{RallyFilter: opts.RallyId != nil})
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
