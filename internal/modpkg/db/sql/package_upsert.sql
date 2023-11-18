-- Upsert packages into the database, marking them to be kept since still in
-- use.
INSERT INTO 
  package (
    module_id, 
    relative_path
  )
SELECT
  $module_id as module_id,
  substr($import_path, length(module.import_path) + 1) as relative_path
FROM
  module
WHERE
  module.rowid = $module_id
ON CONFLICT 
  DO UPDATE SET
    keep = TRUE
RETURNING
  rowid;
