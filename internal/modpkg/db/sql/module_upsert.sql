-- Upsert modules into the database, marking them to be kept since still in
-- use, and marking the module's packages for sync if they may have changed.
INSERT INTO module (
  import_path, 
  version
)
VALUES
  $import_path,
  $version
ON CONFLICT (
  import_path
) DO 
UPDATE SET
  sync = ( 
      excluded.version == ""      -- sync if the version is empty
    OR
      version != excluded.version -- or if the version has changed
  ),
  -- Keep this module since it's still in use.
  keep = TRUE,
  version = excluded.version
RETURNING
  rowid,
  sync
;---
