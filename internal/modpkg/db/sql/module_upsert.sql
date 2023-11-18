-- Upsert modules into the database, marking them to be kept since still in
-- use, and marking the module's packages for sync if they may have changed.
INSERT INTO
  module (
    import_path, 
    relative_dir,
    parent_dir_id,
  )
  SELECT
    $import_path as import_path,
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
      relative_dir != excluded.relative_dir 
    OR 
      parent_dir_id != excluded.parent_dir_id
  ),
  -- Keep this module since it's still in use.
  keep = TRUE,
  (
    relative_dir,
    parent_dir_id
  ) = (
    excluded.relative_dir,
    excluded.parent_dir_id
  )
RETURNING
  rowid,
  sync;
