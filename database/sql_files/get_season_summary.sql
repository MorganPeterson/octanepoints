-- get_season_summary.sql

WITH
  rally_stats AS (
    SELECT
      ro.user_name,
      ro.nationality,
      COUNT(DISTINCT ro.rally_id)                             AS rallies_started,
      SUM(CASE WHEN CAST(ro.position AS INTEGER) = 1 THEN 1 
               ELSE 0 END)                                    AS rally_wins,
      SUM(CASE WHEN CAST(ro.position AS INTEGER) <= 3 THEN 1 
               ELSE 0 END)                                    AS podiums,
      MIN(CAST(ro.position AS INTEGER))                       AS best_position,
      AVG(CAST(ro.position AS INTEGER))                       AS average_position
    FROM rally_overalls ro
    GROUP BY ro.user_name, ro.nationality
  ),

  super_rallied AS (
    SELECT
      rs.user_name,
      COUNT(*)                                              AS total_super_rallied_stages
    FROM rally_stages rs
    WHERE rs.super_rally = 1
    GROUP BY rs.user_name
  ),

  stage_wins AS (
    SELECT
      rs2.user_name,
      COUNT(*)                                              AS stage_wins
    FROM rally_stages rs2
    JOIN (
      SELECT
        rally_id,
        stage_num,
        MIN(time3) AS min_time
      FROM rally_stages
      GROUP BY rally_id, stage_num
    ) sw ON rs2.rally_id   = sw.rally_id
        AND rs2.stage_num  = sw.stage_num
        AND rs2.time3      = sw.min_time
    GROUP BY rs2.user_name
  ),

  -- dynamic mapping: we pass a JSON array like "[10,8,6,5,...]" as the first bind parameter
  points_map AS (
    SELECT
      CAST(json_each.key   AS INTEGER) + 1    AS position,
      CAST(json_each.value AS INTEGER)        AS points
    FROM json_each(?)  -- <-- binds your JSON-array string
  )

SELECT
  rs.user_name,
  rs.nationality,
  rs.rallies_started,
  rs.rally_wins,
  rs.podiums,
  rs.best_position,
  rs.average_position,
  COALESCE(sr.total_super_rallied_stages, 0)   AS total_super_rallied_stages,
  COALESCE(sw.stage_wins,               0)     AS stage_wins,
  COALESCE((
    SELECT SUM(pm.points)
    FROM rally_overalls ro2
    JOIN points_map pm
      ON CAST(ro2.position AS INTEGER) = pm.position
    WHERE ro2.user_name = rs.user_name
  ), 0)                                        AS total_championship_points

FROM rally_stats rs
LEFT JOIN super_rallied sr ON sr.user_name = rs.user_name
LEFT JOIN stage_wins   sw ON sw.user_name = rs.user_name
ORDER BY rs.user_name;
