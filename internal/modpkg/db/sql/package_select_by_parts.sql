WITH RECURSIVE 
  matches (
    remaining_path, 
    part_id, 
    path_depth, 
    total_part_length
  )
AS 
  (
    VALUES (
      $search_path || '/', 
      NULL, 
      0, 
      0
    )
    UNION
    SELECT 
      substr(remaining_path, instr(remaining_path, '/')+1) AS remaining_path,
      part.rowid AS part_id,
      part.path_depth AS path_depth,
      total_part_length + length(part.name) AS total_part_length
    FROM 
      matches, 
      part
    WHERE 
      remaining_path != ''
    AND 
      (
        part.parent_id = matches.part_id
      OR 
        matches.part_id IS NULL
      ) 
    AND 
      (
        (
          $exact IS TRUE
        AND 
          part.name = substr(remaining_path, 1, instr(remaining_path, '/')-1)
        ) 
      OR 
        (
          $exact IS FALSE
        AND
          name LIKE substr(remaining_path, 1, instr(remaining_path, '/')-1) || '%' 
        )
      )
    ORDER BY 
      3 DESC, 
      4 ASC
  )
SELECT 
  package_import_path,
  dir
FROM 
  matches, part_package USING (part_id), 
  package_view USING (package_id) 
WHERE 
  remaining_path = ''
ORDER BY 
  (total_num_parts - path_depth) ASC,
  total_part_length ASC,
  total_num_parts ASC
;
