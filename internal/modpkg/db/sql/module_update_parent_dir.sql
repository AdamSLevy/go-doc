UPDATE
  module
SET
  parent_dir_id = ( -- set the parent dir id to
    iif(
      $vendor,
      $parent_dir_id_vendor, -- the vendor parent dir id if switching _to_ vendor mode
      $parent_dir_id_gomodcache, -- the mod cache dir id if switching _from_ vendor mode
    )
  )
WHERE
  parent_dir_id = ( -- only for modules with the parent dir id set to
    iif(
      NOT $vendor, 
      $parent_dir_id_vendor, -- the vendor parent dir if switching _from_ vendor mode
      $parent_dir_id_gomodcache, -- the mod cache dir if switching _to_ vendor mode
    )
  );
