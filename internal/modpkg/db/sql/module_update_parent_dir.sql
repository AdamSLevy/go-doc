UPDATE
  module
SET
  parent_dir_id = (
    iif(
      $1,
      $2,
      $3
    )
  )
WHERE
  parent_dir_id = (
    iif(
      NOT $1,
      $2,
      $3
    )
  )
;
