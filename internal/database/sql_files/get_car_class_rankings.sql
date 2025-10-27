-- sql/get_rankings.sql

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
  JOIN cars       c  ON c.id     = ro.car_id
  JOIN class_cars cc ON cc.car_id = c.id

  -- this single WHERE does “no filter” when ? IS NULL,
  -- or “only rally = ?” when you pass a number
  WHERE ro.rally_id = COALESCE(?, ro.rally_id)
)

SELECT
  rally_id,
  class_id,
  user_id,
  user_name,
  time3,
  pos
FROM ranked
ORDER BY rally_id, class_id, pos;
