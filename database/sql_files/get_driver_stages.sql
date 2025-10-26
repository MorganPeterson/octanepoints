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
    WHERE rally_id   = ?1 
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
WHERE r.user_name = ?2
ORDER BY r.stage_num
