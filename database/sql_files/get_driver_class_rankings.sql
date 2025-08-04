WITH driver_classes AS (
  SELECT
    ro.rally_id,
    ro.user_id,
    ro.user_name,
    ro.time3,
    cd.class_id
  FROM rally_overalls ro
  JOIN class_drivers cd ON cd.user_id = ro.user_id

  -- this single WHERE does “no filter” when ? IS NULL,
  -- or “only rally = ?” when you pass a number
  WHERE ro.rally_id = COALESCE(?, ro.rally_id)
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

