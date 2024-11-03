-- The schema for the database.
--
-- Use ;--- to separate statements. This is a simple hack to allow for
-- splitting complete statements. Omitting the ;--- won't result in an error,
-- but that statement will be executed together with all subsequent statements
-- until the next ;--- or the end of the file.

-- module stores all required modules and the directory they are located in.
--
-- import_path is the module's import path.
--
-- version is the module's version, if any.
--
-- relative_dir is the directory the module is located in, relative to the
-- parent_dir.dir referenced by parent_dir_id.
--
-- parent_dir_id is the parent_dir's rowid.
--
-- num_segments is the number of slash separated parts in the module's import path.
--
-- sync is a boolean that indicates whether the module's packages should be
-- synced. Newly inserted modules have sync set to TRUE. Upserted modules have
-- sync set to true if the module's dir has changed.
--
-- keep is a boolean that indicates whether the module should be kept. At the
-- beginning of a sync, all module's keep are set to FALSE. Newly inserted
-- modules have keep set to TRUE. Upserted modules have keep set to TRUE. After
-- syncing all modules, any modules that have keep set to FALSE are deleted.
CREATE TABLE module (
  rowid INTEGER PRIMARY KEY,

  import_path   TEXT NOT NULL UNIQUE CHECK (
    -- must not have leading or trailing slashes
    import_path = trim(import_path, '/') 
  ),

  version     TEXT NOT NULL,

  num_segments INT NOT NULL GENERATED ALWAYS AS (
    iif(
      length(import_path) = 0, 
      0, 
        1 
      + length(import_path) 
      - length(replace(import_path, '/', ''))
      )
  ) STORED,

  sync   BOOL NOT NULL DEFAULT TRUE,
  keep   BOOL NOT NULL DEFAULT TRUE
);---

-- package stores all packages for all modules.
--
-- module_id is the module the package belongs to.
--
-- relative_path is the package's path relative to the module's import_path.
-- This can be empty if the module's import path is an importable package.
--
-- num_segments is the number of slash separated parts in the package's relative path.
--
-- keep is a boolean that indicates whether the package should be kept.
-- Whenever a module requires a sync, keep is set to FALSE for all of its
-- packages. Newly inserted packages have keep set to TRUE. When existing
-- packages are upserted, keep is set back to TRUE. After syncing all packages,
-- any that have keep set to FALSE are deleted.
CREATE TABLE package (
  rowid         INTEGER PRIMARY KEY,

  module_id     INT     NOT NULL
                        REFERENCES module(rowid)
                          ON DELETE CASCADE
                          ON UPDATE CASCADE,

  in_mod_path TEXT NOT NULL UNIQUE CHECK (
    -- must not have leading or trailing slashes
    in_mod_path = trim(in_mod_path, '/') 
  ),
  UNIQUE(module_id, in_mod_path),

  num_segments INT NOT NULL GENERATED ALWAYS AS (
    iif(
      length(in_mod_path) = 0,
      0,
        1
      + length(in_mod_path)
      - length(replace(in_mod_path, '/', ''))
    )
  ) STORED,

  keep BOOL NOT NULL DEFAULT TRUE
);---

-- module_package is a view that joins module and package information.
--
-- package_id is the package's rowid.
--
-- package_import_path is the package's import path.
--
-- package_dir is the directory the package is located in.
--
-- module_id is the module's rowid.
--
-- module_import_path is the module's import path.
--
-- relative_path is the package's path relative to the module's import_path.
--
-- class is an integer that represents the type of module.
--
-- relative_num_segments is the number of slash separated parts in the package's relative_path.
--
-- total_num_segments is the number of slash separated parts in the package_import_path.
CREATE VIEW module_package (
  rowid,      -- package.rowid
  package_id, -- package.rowid
  import_path,
  total_num_segments,
  module_id,
  module_import_path,
  module_num_segments
) AS 
SELECT
  package.rowid         AS rowid,
  package.rowid         AS package_id,
  concat_ws(
    '/', 
    module.import_path, 
    package.in_mod_path 
  )                     AS import_path,
  package.num_segments +
    module.num_segments AS total_num_segments,
  module.rowid          AS module_id,
  module.import_path    AS module_path,
  module.num_segments   AS module_num_segments
FROM
  package, 
  module
ON
  package.module_id = module.rowid
ORDER BY
  module.num_segments  ASC,
  module.import_path   ASC,
  package.num_segments ASC,
  package.in_mod_path  ASC
;---
