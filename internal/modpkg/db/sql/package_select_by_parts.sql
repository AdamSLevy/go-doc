WITH RECURSIVE 
  matches (
    remaining_path, 
    part_id, 
    path_depth, 
    part_length
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
      substr(remaining_path, instr(remaining_path, '/')+1),
      part.rowid,
      part.path_depth,
      part_length + length(part.name)
    FROM matches, part
    WHERE
      name LIKE substr(remaining_path, 1, instr(remaining_path, '/')-1) || '%' 
    AND (
        part.parent_id = matches.part_id
      OR 
        matches.part_id IS NULL
    )
    AND
      remaining_path != ''
    ORDER BY 
      3 DESC, 
      4 ASC
  )
SELECT 
  package_id, 
  package_import_path,
  dir
FROM 
  matches, part_package USING (part_id), 
  package_view USING (package_id) 
WHERE 
  remaining_path = ''
ORDER BY 
  (total_num_parts - path_depth) ASC,
  total_num_parts ASC,
  part_length ASC
;
