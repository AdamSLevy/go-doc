WITH RECURSIVE
split_search_path(
  start_pos, 
  end_pos,
  prev_part,
  prev_path_depth,
  remaining
) AS (
  SELECT 
    1                              AS start_pos, 
    0                              AS end_pos,
    ''                             AS prev_part,
    0                              AS prev_path_depth,
    trim($search_path, '/') || '/' AS remaining
  UNION ALL
  SELECT 
    end_pos + 1 AS start_pos, 
    instr(
      remaining,
      '/'
    ) AS end_pos,

    iif(end_pos > 0,
      substr(remaining, start_pos, end_pos),
      ''
    ) AS prev_part,

    iif(end_pos > 0,
      prev_path_depth + 1,
      0
    ) AS prev_path_depth,

    iif(end_pos > 0,
      substr(remaining, end_pos + 1),
      remaining
    ) AS remaining
  FROM 
    split_search_path
  WHERE 
    remaining != ''
),

search AS (
  SELECT 
    prev_part || iif($exact, '', '%') AS name,
    prev_path_depth                   AS path_depth,
    count(*) - prev_path_depth        AS min_deepest_descendant_distance
  FROM
    split_search_path
  WHERE
    prev_part != ''
),

match_search (
  first_id,
  part_id,
  search_path_depth
) AS (
  SELECT
    part.rowid        AS first_id,
    part.rowid        AS part_id,
    search.path_depth AS search_path_depth
  FROM
    part_view   AS part, 
    search
  ON 
    part.name LIKE search.name
  WHERE
    search.path_depth = 1
  AND
    part.deepest_descendant_distance >= search.min_deepest_descendant_distance
  UNION ALL
  SELECT
    match.first_id    AS first_id,
    part.rowid        AS part_id,
    search.path_depth AS search_path_depth
  FROM
    part_view AS part,
    match_search AS match,
    search 
  ON
    part.parent_id = match.part_id
  AND
    part.name LIKE search.name
  AND
    part.max_descendant_distance >= search.min_max_descendant_distance 
  AND
    search.path_depth = match.search_path_depth + 1
),

match AS (
  SELECT
    first_id,
    part_id
  FROM
    match_search
  WHERE
    search_path_depth = (
      SELECT 
        count(*)
      FROM 
        search
    )
),

package_match AS (
  SELECT
    match.first_id,
    match.part_id
  FROM
    match,
    part
  ON
    part.rowid = match.part_id
  WHERE
    part.package_id IS NOT NULL
),

dir_match (
  first_id,
  part_id,
  package_id,
  num_children
) AS (
  SELECT
    match.first_id    AS first_id,
    match.part_id     AS part_id,
    part.package_id   AS package_id,
    part.num_children AS num_children,
    0                 AS unambiguous_depth
  FROM
    match,
    part
  ON
    part.rowid = match.part_id
  WHERE
    part.num_children > 0
  UNION ALL
  SELECT
    dir_match.first_id AS first_id,
    part.rowid         AS part_id,
    part.package_id    AS package_id,
    part.num_children  AS num_children,
    dir_match.unambiguous_depth + 1 AS unambiguous_depth
  FROM
    dir_match,
    part
  ON
    part.parent_id = dir_match.part_id
  WHERE
    dir_match.num_children = 1
  AND (
      dir_match.package_id IS NULL
    OR
      unambiguous_depth = 0
    )
),

follow_unambiguous AS (
  SELECT
    match.first_id,
    match.part_id
  FROM
    dir_match,
    part
  ON
    part.rowid = dir_match.part_id
  WHERE
    part.num_children = 1
  UNION ALL
  SELECT
    follow_unambiguous.first_id,
    part.rowid
  FROM
    follow_unambiguous,
    part
  ON
    part.parent_id = follow_unambiguous.part_id
  WHERE
    part.package_id IS NOT NULL
  OR
    part.num_children = 1
),

descend_dir_math AS (
  SELECT
    deepest_match.first_id,
    deepest_match.part_id
  FROM
    deepest_match,
    part
  ON
    part.rowid = deepest_match.part_id
  WHERE
    part.package_id IS NULL
  AND
    part.num_children = 1
  UNION ALL
  SELECT
    descend_dir_match.first_id AS first_id,
    part.rowid AS part_id
  FROM
    descend_dir_match,
    part
  ON
    part.parent_id = descend_dir_match.part_id
  WHERE
 
    part.package_id IS NULL
),




unambiguous_child_match(
  first_id,
  part_id
) AS (
  SELECT 
    dir_match.first_id,
    dir_match.part_id
  FROM 
    dir_match,
    part
  ON 
    part.rowid = dir_match.part_id
  WHERE
    part.num_children = 1
  UNION ALL
  SELECT
    unambiguous_child_match.first_id,
    part.rowid
  FROM
    unambiguous_child_match,
    part
  ON
    part.parent_id = unambiguous_child_match.part_id
  WHERE
    part.package_id IS NOT NULL
  OR
    part.num_children = 1

build_full_path(part_id, full_path) AS (
  -- Build full path from root to deepest matching parts
  SELECT dm.part_id, dm.matched_path
  FROM deepest_matches dm
  WHERE dm.parent_id IS NULL
  UNION ALL
  SELECT p.rowid, concat_ws('/', bfp.full_path, p.name)
  FROM part p
  JOIN build_full_path bfp ON p.rowid = bfp.part_id
),
descend_unambiguous(part_id, parent_id, first_matched_parent_id, depth, is_package, num_children, matched_path) AS (
  -- Descend to unambiguous parts
  SELECT dm.part_id, dm.parent_id, dm.first_matched_parent_id, dm.depth, dm.is_package, dm.num_children, bfp.full_path
  FROM deepest_matches dm
  JOIN build_full_path bfp ON dm.part_id = bfp.part_id
  UNION ALL
  SELECT p.rowid, p.parent_id, du.first_matched_parent_id, du.depth + 1, p.package_id IS NOT NULL, p.num_children,
         concat_ws('/', du.matched_path, p.name)
  FROM part p
  JOIN descend_unambiguous du ON p.parent_id = du.part_id
  WHERE du.num_children = 1
)
SELECT ...
FROM descend_unambiguous
JOIN ... -- Join with package or module tables
WHERE ... -- Final conditions


