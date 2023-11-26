-- Upsert modules into the database, marking them to be kept since still in
-- use, and marking the module's packages for sync if they may have changed.
INSERT INTO
  module (
    import_path, 
    version,
    relative_dir,
    parent_dir_id
  )
  SELECT
    $import_path as import_path,
    $version as version,
    substr($dir, length(parent_dir.dir) + 1) as relative_dir,
    parent_dir.rowid as parent_dir_id
  FROM
    parent_dir
  WHERE
    instr($dir, parent_dir.dir) == 1
ON CONFLICT (
  import_path
)
DO UPDATE SET
  -- Only sync if the relative or parent dir id has changed.
  sync = ( 
      excluded.version == "" -- always sync if the version is empty
    OR
      version != excluded.version -- or if the version has changed
  ),
  -- Keep this module since it's still in use.
  keep = TRUE,
  (
    version,
    relative_dir,
    parent_dir_id
  ) = (
    excluded.version,
    excluded.relative_dir,
    excluded.parent_dir_id
  )
RETURNING
  sync,
  rowid,
  relative_dir,
  parent_dir_id
;
