WITH RECURSIVE 
  matches (
    matched_path,
    remaining_path, 
    part_id, 
    part_path_depth
  )
AS 
  (
    VALUES (
      '',
      $search_path || '/', 
      NULL, 
      0, 
      0
    )
    UNION
    SELECT 
      concat_ws('/', matched_path, part.name)              AS matched_path,
      substr(remaining_path, instr(remaining_path, '/')+1) AS remaining_path,
      part.rowid                                           AS part_id,
      part.path_depth                                      AS part_path_depth
    FROM 
      matches, 
      part
    WHERE 
      remaining_path != '' -- we still have parts to match
      AND 
      (
        matches.part_id IS NULL          -- handles initial case where no parts have been matched
        OR
        part.parent_id = matches.part_id -- the part is a child of a previously matched part
      ) 
      AND 
      ( -- the part name matches the next part in the path
        (
          $exact IS TRUE
          AND 
          part.name = substr(remaining_path, 1, instr(remaining_path, '/')-1) -- match exactly
        ) 
        OR 
        name LIKE substr(remaining_path, 1, instr(remaining_path, '/')-1) || '%'  -- match prefix
      )
    ORDER BY 
      part_path_depth   DESC, 
      length(matched_path) ASC
  )
SELECT 
  package_import_path,
  dir,
  matched_path
FROM 
  matches, part_package USING (part_id), 
  package_view USING (package_id) 
WHERE 
  remaining_path = ''
ORDER BY 
  (total_num_parts - part_path_depth) ASC,
  length(matched_path) ASC,
  total_num_parts ASC
;
