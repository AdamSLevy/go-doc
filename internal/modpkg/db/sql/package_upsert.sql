-- Upsert packages into the database, marking them to be kept since still in
-- use.
INSERT INTO 
  package (
    module_id, 
    relative_path
  )
VALUES (
  $module_id,
  $relative_path
)
ON CONFLICT(module_id, relative_path)
  DO UPDATE SET
    keep = TRUE
RETURNING
  rowid;
