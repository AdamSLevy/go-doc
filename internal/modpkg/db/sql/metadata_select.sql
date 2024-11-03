SELECT 
  created_at, 
  updated_at, 

  build_revision, 
  go_version,

  go_mod_hash,
  go_sum_hash,
  vendor
FROM 
  metadata 
WHERE
  rowid = 1
LIMIT 1;
