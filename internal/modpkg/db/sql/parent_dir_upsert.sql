INSERT INTO
  parent_dir (
    rowid,
    key,
    dir
  )
VALUES (
    $rowid, $key, $dir
)
ON CONFLICT
DO UPDATE SET
  (
    rowid, 
    key, 
    dir
  ) = (
    excluded.rowid,
    excluded.key,
    excluded.dir
  )
;
