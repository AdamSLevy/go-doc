
INSERT INTO 
  metadata(
    rowid, 

    build_revision, 
    go_version,

    go_mod_hash,
    go_sum_hash,
    vendor
  ) 
VALUES (
  1 as rowid,

  $build_revision as build_revision,
  $go_version as go_version,

  $go_mod_hash as go_mod_hash,
  $go_sum_hash as go_sum_hash,
  $vendor as vendor
)
ON CONFLICT(rowid) DO 
  UPDATE SET 
    updated_at = CURRENT_TIMESTAMP, 
    (
      build_revision,
      go_version,
 
      go_mod_hash,
      go_sum_hash,
      vendor
    ) = (
      excluded.build_revision,
      excluded.go_version,

      excluded.go_mod_hash,
      excluded.go_sum_hash,
      excluded.vendor
    );
