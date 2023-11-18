SELECT 
  rowid, 
  name, 
  parent_id, 
  package_id 
FROM 
  part 
WHERE 
  rowid IN ( 
    SELECT 
      part_id 
    FROM 
      part_package
    WHERE 
      package_id = ?
  ) 
ORDER BY 
  path_depth ASC;
