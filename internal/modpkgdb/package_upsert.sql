-- Upsert packages into the database, marking them to be kept since still in
-- use.
INSERT INTO 
  package (
    module_id, 
    relative_path
  )
VALUES (
  ?, ?
)
ON CONFLICT 
  DO UPDATE SET
    keep = TRUE
RETURNING
  rowid;
