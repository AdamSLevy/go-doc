-- The schema for the database.
--
-- Use ;--- to separate statements. This is a simple hack to allow for
-- splitting complete statements. Omitting the ;--- won't result in an error,
-- but that statement will be executed together with all subsequent statements
-- until the next ;--- or the end of the file.

-- metadata stores information about the database.
--
-- rowid is the primary key and is always 1 to ensure there is only one row.
--
-- created_at is the time the database was created.
--
-- updated_at is the time the database was last updated.
--
-- build_revision is the git revision of the go-doc build which last updated
-- this database.
--
-- go_version is the version of Go used to build the go-doc binary.
--
-- go_root is the path to the Go root directory.
--
-- go_mod_cache is the path to the Go module cache.
--
-- main_mod_id is the module_id of the main module.
--
-- go_mod_hash is the CRC32 hash of the go.mod file.
--
-- go_sum_hash is the CRC32 hash of the go.sum file.
--
-- vendor is a boolean that indicates whether the main module is vendored.
CREATE TABLE metadata (
  rowid INTEGER PRIMARY KEY 
        NOT NULL 
        CHECK (rowid = 1),

  created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  build_revision TEXT NOT NULL CHECK (build_revision != ''),
  go_version     TEXT NOT NULL CHECK (go_version     != ''),

  main_mod_dir TEXT NOT NULL CHECK (main_mod_dir != ''),
  go_mod_hash  INT  NOT NULL CHECK (go_mod_hash != 0),
	go_sum_hash  INT  NOT NULL CHECK (go_sum_hash != 0),
  vendor       BOOL NOT NULL DEFAULT FALSE
) WITHOUT ROWID;---
