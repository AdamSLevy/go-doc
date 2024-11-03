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

  go_mod_hash INT  NOT NULL CHECK (go_mod_hash != 0),
	go_sum_hash INT  NOT NULL CHECK (go_sum_hash != 0),
  vendor      BOOL NOT NULL DEFAULT FALSE
) WITHOUT ROWID;---

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
  go_sum_hash TEXT NOT NULL,

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

-- package_view is a view that joins module and package information.
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
CREATE VIEW package_view (
  rowid,
  import_path,
  num_segments,
  module_id,
  module_import_path,
  module_num_segments
) AS SELECT
  package.rowid         AS rowid,
  concat_ws(
    '/', 
    module.import_path, 
    package.in_mod_path 
  )                     AS import_path,
  package.num_segments  AS num_segments,
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

CREATE TABLE import_path_segment_name (
  rowid INTEGER PRIMARY KEY,
  name  TEXT UNIQUE NOT NULL CHECK (name != '')
);---

CREATE TABLE import_path_segment (
  rowid INTEGER PRIMARY KEY,
  parent_id INT NOT NULL
                REFERENCES import_path_segment(rowid)
                  ON DELETE CASCADE
                  ON UPDATE RESTRICT,
  name_id    INT NOT NULL
                REFERENCES import_path_segment_name(rowid)
                  ON DELETE RESTRICT
                  ON UPDATE CASCADE,
  UNIQUE(parent_id, name_id),

  path_depth INT NOT NULL CHECK (path_depth > 0),
  package_id INT UNIQUE
                REFERENCES package(rowid)
                  ON DELETE SET NULL
                  ON UPDATE CASCADE
);---

CREATE VIEW import_path_segment_view (
  rowid,
  parent_id,
  name,
  path_depth,
  package_id,
  num_children,
  num_descendants,
  max_descendant_distance,
) AS
SELECT
  segment.rowid      AS rowid,
  segment.parent_id  AS parent_id,
  segmant_name.name  AS name,
  segment.path_depth AS path_depth,
  segment.package_id AS package_id,
  closure.num_children,
  closure.num_descendants,
  closure.max_descendant_distance
FROM
  import_path_segment         AS segment
JOIN
  import_path_segment_name    AS segment_name
ON
  segment.name_id = segment_name.rowid
JOIN
  import_path_segment_closure_view AS closure
ON
  closure.segment_id = segment.rowid
;---

CREATE TABLE import_path_segment_package (
  package_id INT NOT NULL
                REFERENCES package(rowid)
                  ON DELETE CASCADE
                  ON UPDATE CASCADE,
  segment_id INT NOT NULL
                REFERENCES import_path_segment(rowid)
                  ON DELETE CASCADE
                  ON UPDATE CASCADE,
  PRIMARY KEY(package_id, segment_id),

  path_depth INT NOT NULL CHECK (path_depth > 0),
  UNIQUE(package_id, path_depth)
) WITHOUT ROWID;---

CREATE TABLE import_path_segment_closure (
  ancestor_id   INT NOT NULL
                  REFERENCES import_path_segment(rowid)
                    ON DELETE CASCADE
                    ON UPDATE CASCADE,
  descendant_id INT NOT NULL
                  REFERENCES import_path_segment(rowid)
                    ON DELETE CASCADE
                    ON UPDATE CASCADE,
  PRIMARY KEY(ancestor_id, descendant_id),
  distance INT NOT NULL CHECK (distance >= 0)
) WITHOUT ROWID;---

CREATE VIEW import_path_segment_closure_view (
  segment_id,
  num_children,
  num_descendants,
  max_descendant_distance
) 
AS SELECT
  closure.ancestor_id            AS segment_id,
  count(closure.descendant_id) 
    FILTER ( 
      WHERE closure.distance = 1 
    )                            AS num_children,
  count(closure.descendant_id)   AS num_descendants,
  max(closure.distance)          AS descendant_max_distance
FROM
  import_path_segment_closure AS closure
GROUP BY
  closure.ancestor_id
;---
 

-- import_path_segment_split_view is a view that exists purely to allow for an INSTEAD OF
-- trigger to be used, which automatically splits a package path into parts.
CREATE VIEW 
  import_path_segment_split_view (
    package_id,
    total_num_segments,
    path_depth,
    segment_parent_id,
    segment_name,
    remaining_path
  ) 
AS 
VALUES (
  NULL,
  NULL,
  NULL,
  NULL,
  NULL,
  NULL
);---

-- import_path_segment_splitter_on_insert_package is a trigger that fires whenever
-- a new package is inserted.
--
-- This initiates a recursive trigger chain that splits the package's path and
-- inserts them into the part table.
CREATE TRIGGER 
  import_path_segment_splitter_on_insert_package
AFTER 
  INSERT ON 
    package
BEGIN
  INSERT INTO 
    import_path_segment_split_view (
      package_id,
      total_num_segments,
      path_depth,
      segment_parent_id,
      segment_name,
      remaining_path
    )
    SELECT
      package.rowid              AS package_id,
      package.num_segments + 
        module.num_segments      AS total_num_segments,
      0                          AS path_depth,
      NULL                       AS segment_parent_id,
      ''                         AS segment_name,
      package.import_path || '/' AS remaining_path
    FROM
      package, module
    ON 
      package.module_id = module.rowid
    WHERE
      package.rowid = new.rowid;
END;---

-- recursive_package_part_splitter is a recursive trigger that fires instead of
-- inserting to the import_path_segment_split_view view.
--
-- It inserts the current part into the part and part_package tables.
--
-- It then inserts the next part, if any, into the import_path_segment_split_view view.
CREATE TRIGGER 
  recursive_package_part_splitter
INSTEAD OF 
  INSERT ON 
    import_path_segment_split_view
  WHEN
    new.path_depth <= new.total_num_segments
BEGIN

  -- insert the segment name
  INSERT INTO
    import_path_segment_name (
      name
    )
  SELECT
    new.segment_name
  WHERE
    new.path_depth > 0
  ON CONFLICT DO
    NOTHING;

  -- insert the segment
  INSERT INTO
    import_path_segment (
      parent_id,
      name_id,
      path_depth,
      package_id
    )
  SELECT
    new.segment_parent_id          AS parent_id,
    -- if the segment name was inserted, changes() will be 1, so we can use
    -- last_insert_rowid() and avoid the subquery
    iif(
      changes() > 0, 
      last_insert_rowid(),
      (
        SELECT 
          rowid 
        FROM 
          import_path_segment_name 
        WHERE 
          name IS new.segment_name
      )
    )                              AS name_id,
    new.path_depth                 AS path_depth,

    -- the package_id is only set on the final segment, otherwise it is NULL
    iif(
      new.path_depth = new.total_num_segments,
      new.package_id,
      NULL
    )                              AS package_id
  WHERE
    new.path_depth > 0
  ON CONFLICT DO
    UPDATE SET
      package_id = excluded.package_id
    WHERE
      new.path_depth = new.total_num_segments;

  INSERT INTO
    import_path_segment_split_view (
      package_id,
      total_num_segments,
      path_depth,
      segment_parent_id,
      segment_name,
      remaining_path
    )
  SELECT
    new.package_id         AS package_id,
    new.total_num_segments AS total_num_segments,
    new.path_depth+1       AS path_depth,

    -- the first iteration will have NULL segment_parent_id
    iif(
      new.path_depth = 0,
      NULL,

      -- if the segment was inserted, changes() will be 1, so we can avoid the
      -- subquery and use last_insert_rowid()
      iif(
        changes() > 0,
        last_insert_rowid(),
        (
          SELECT 
            rowid 
          FROM 
            import_path_segment_view
          WHERE 
            parent_id IS new.segment_parent_id
          AND 
            name IS new.segment_name
        )
      )
    ) AS segment_parent_id,

    substr(new.remaining_path, 1, slash-1) AS segment_name,
    substr(new.remaining_path, slash+1)    AS remaining_path
  FROM (
    -- find the position of the first slash
    SELECT
      instr(new.remaining_path, '/') AS slash
  )
  WHERE
    -- when new.path_depth = new.total_num_segments, we have inserted all
    -- segments and we are done
    new.path_depth < new.total_num_segments;

END;---

-- insert_part_path_closure is a trigger that fires whenever a new part is
-- inserted into the part table. It populates the part_path table with the new
-- part and all of its ancestors.
CREATE TRIGGER 
  build_import_path_segment_closure
AFTER 
  INSERT ON 
    import_path_segment
BEGIN
 
  -- insert all ancestors of the new segment, including the segment itself
  INSERT INTO 
    import_path_segment_closure (
      ancestor_id,
      descendant_id,
      distance
    )
  -- the new segment is its own ancestor, with a distance of 0
  SELECT
    new.rowid AS ancestor_id, 
    new.rowid AS descendant_id, 
    0         AS distance
  UNION ALL
  -- all ancestors of the new segment's parent, are also ancestors of the new
  -- segment with a distance of 1 more than the distance to the parent
  SELECT 
    closure.ancestor_id  AS ancestor_id, 
    new.rowid            AS descendant_id,
    closure.distance + 1 AS distance
  FROM 
    import_path_segment_closure AS closure
  WHERE
    closure.ancestor_id IS new.parent_id;

END;---

-- prune_leaf_parts_with_null_package_id is a trigger that fires whenever
-- a part is updated such that its package_id is set to NULL. It deletes the
-- part if it has no children.
CREATE TRIGGER 
  prune_leaf_segments_with_null_package_id
AFTER 
  UPDATE OF 
    package_id 
  ON 
    import_path_segment
  WHEN 
    new.package_id IS NULL
BEGIN

  DELETE FROM
    import_path_segment
  WHERE
    rowid = new.rowid
  AND
    NOT EXISTS (
      SELECT
        1
      FROM
        import_path_segment AS segment
      WHERE
        segment.parent_id IS new.rowid
    );

END;---

-- recursively_prune_leaf_parts_with_null_package_id recursively deletes leaf
-- parts that have a NULL package_id.
CREATE TRIGGER 
  recursively_prune_leaf_parts_with_null_package_id
AFTER 
  DELETE ON 
    import_path_segment
BEGIN

  DELETE FROM
    import_path_segment
  WHERE
    package_id IS NULL
  AND
    NOT EXISTS (
      SELECT
        1
      FROM
        part AS p
      WHERE
        p.parent_id IS part.rowid
    );

END;---

-- set_package_keep_false_for_modules_with_sync_true fires whenever a module is
-- updated such that sync is set to TRUE. It sets package.keep to FALSE for all
-- of the module's packages. As packages are re-synced, package.keep is set
-- back to TRUE so that only packages that are no longer available in the
-- module are left with keep set to FALSE, resulting in them being deleted to
-- finalize a sync.
CREATE TRIGGER
  set_package_keep_false_for_modules_with_sync_true
AFTER
  UPDATE OF
    sync
  ON 
    module
  WHEN
    new.sync = TRUE
BEGIN
  UPDATE
   package
  SET
    keep = FALSE
  WHERE
    package.module_id = new.rowid;
END;---

-- after_update_metadata_prune_module_package fires after the metadata has been
-- updated. It performs some sanity checks to ensure that the sync was
-- completed successfully and then deletes all modules and packages that have
-- keep set to FALSE.
CREATE TRIGGER
  after_update_metadata_prune_module_package
AFTER
  UPDATE ON
    metadata
BEGIN
  SELECT
    RAISE(ABORT, 'invalid sync: no modules were synced')
  WHERE
    NOT EXISTS (
      SELECT
        1
      FROM
        module
      WHERE
        keep = TRUE
    );

  SELECT
    RAISE(ABORT, 'invalid sync: no packages were synced for one or more modules')
  WHERE
    EXISTS (
      SELECT
        1
      FROM (
        SELECT
          count(*) AS num_pkgs
        FROM 
          package
        WHERE
          keep = TRUE
        GROUP BY
          module_id
      )
      WHERE
        num_pkgs = 0
    );

  DELETE FROM
    module
  WHERE
    keep = FALSE;

  DELETE FROM
    package
  WHERE
    keep = FALSE;

END;---
