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
-- build_revision is the git revision of the build.
--
-- go_version is the version of Go used to build the binary.
CREATE TABLE metadata (
  rowid          INTEGER  PRIMARY KEY CHECK (rowid = 1),
  created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  build_revision TEXT     NOT NULL CHECK (build_revision != ''),
  go_version     TEXT     NOT NULL CHECK (go_version != '')
) WITHOUT ROWID;---

-- module stores all required modules and the directory they are located in.
--
-- import_path is the module's import path.
--
-- dir is the directory the module is located in.
--
-- class is an integer that represents the type of module:
--  0: standard library
--  1: local/main module
--  2: required module
--
-- vendor is a boolean that indicates whether the module is a vendored module.
--
-- num_parts is the number of slash separated parts in the module's import path.
CREATE TABLE module (
  rowid       INTEGER PRIMARY KEY,
  import_path TEXT    UNIQUE NOT NULL,
  dir         TEXT    NOT NULL CHECK (dir != ''),
  class       INT     NOT NULL CHECK (class >= 0 AND class <= 2),
  vendor      BOOL    DEFAULT false,
  num_parts   INT     GENERATED ALWAYS AS (
      length(trim(import_path, '/'))
      - length(replace(trim(import_path, '/'), '/', ''))
      + iif(trim(import_path, '/') = '', 0, 1)
    ) STORED,

  sync   BOOL DEFAULT TRUE,
  keep   BOOL DEFAULT TRUE
);---

CREATE INDEX module_class ON module(class, import_path);---

-- package stores all packages for all modules.
--
-- module_id is the module the package belongs to.
--
-- relative_path is the package's path relative to the module's import_path.
-- This can be empty if the module's import path is an importable package.
--
-- num_parts is the number of slash separated parts in the package's relative path.
CREATE TABLE package (
  rowid         INTEGER PRIMARY KEY,
  module_id     INT     NOT NULL
                        REFERENCES module(rowid)
                          ON DELETE CASCADE
                          ON UPDATE CASCADE,
  relative_path TEXT    NOT NULL,
  num_parts     INT     GENERATED ALWAYS AS (
      length(trim(relative_path, '/'))
      - length(replace(trim(relative_path, '/'), '/', ''))
      + iif(trim(relative_path, '/') = '', 0, 1)
    ) STORED,

  keep BOOL DEFAULT TRUE,

  UNIQUE(module_id, relative_path)
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
-- vendor is a boolean that indicates whether the package is in a vendored
-- module.
--
-- relative_num_parts is the number of slash separated parts in the package's relative_path.
--
-- total_num_parts is the number of slash separated parts in the package_import_path.
CREATE VIEW 
  module_package (
    package_id,
    package_import_path,
    package_dir,
    module_id,
    module_import_path,
    relative_path,
    class,
    vendor,
    relative_num_parts,
    total_num_parts
  )
AS SELECT
  package.rowid AS package_id,
  trim(module.import_path || '/' || package.relative_path, '/') AS package_import_path,
  rtrim(module.dir        || '/' || package.relative_path, '/') AS package_dir,
  package.module_id AS module_id,
  module.import_path AS module_import_path,
  relative_path,
  class,
  vendor,
  package.num_parts AS relative_num_parts,
  package.num_parts + module.num_parts AS total_num_parts
FROM
  package, module
ON
  package.module_id=module.rowid
ORDER BY
  class              ASC,
  module_import_path ASC,
  relative_num_parts ASC,
  relative_path      ASC;---

-- part is a directed acyclic graph of all slash separated parts of all package
-- import paths. It is used to implement searching for packages by partial
-- paths.
--
-- parent_id is the parent part. This is NULL for root parts.
--
-- name is the part's name.
--
-- package_id is the package the part belongs to. This is not NULL for leaf
-- parts only.
--
-- path_depth is the number of parts in the part's path, including itself.
CREATE TABLE part (
  rowid      INTEGER PRIMARY KEY,
  parent_id  INT     REFERENCES part(rowid)
                       ON DELETE CASCADE
                       ON UPDATE CASCADE,
  name       TEXT    NOT NULL CHECK (name != ''),
  package_id INT     UNIQUE 
                     REFERENCES package(rowid)
                       ON DELETE SET NULL
                       ON UPDATE CASCADE,
  path_depth INT     NOT NULL CHECK (path_depth > 0),

  UNIQUE(parent_id, name)
);---

CREATE        INDEX part_idx_name           ON part(name);---
CREATE        INDEX part_idx_package_id     ON part(package_id);---
CREATE UNIQUE INDEX part_idx_parent_id_name ON part(parent_id, name) WHERE parent_id IS NOT NULL;---
CREATE UNIQUE INDEX part_idx_root_name      ON part(name)            WHERE parent_id IS NULL;---

-- part_package is a many-to-many relationship between part and package. This
-- can be used to find all packages that contain a part, or all parts that make
-- up a package's import path.
--
-- part_id is the part's rowid.
--
-- package_id is the package's rowid.
CREATE TABLE part_package (
  part_id    INT NOT NULL
                 REFERENCES part(rowid)
                   ON DELETE CASCADE
                   ON UPDATE CASCADE,
  package_id INT NOT NULL
                 REFERENCES package(rowid)
                   ON DELETE CASCADE
                   ON UPDATE CASCADE,
  PRIMARY KEY(part_id, package_id)
) WITHOUT ROWID;---

CREATE INDEX part_package_idx_package_id ON part_package(package_id, part_id);---

-- part_path is a transitive closure of part, relating all parts to all of
-- their anscestors and descendants. A part is an ancestor and descendent of
-- itself.
--
-- descendant_id is the descendant part's rowid.
--
-- ancestor_id is the ancestor part's rowid.
--
-- distance is the number of parts between the descendant and ancestor.
CREATE TABLE part_path (
  descendant_id INT NOT NULL
                    REFERENCES part(rowid)
                      ON DELETE CASCADE
                      ON UPDATE CASCADE,
  ancestor_id   INT NOT NULL
                    REFERENCES part(rowid)
                      ON DELETE CASCADE
                      ON UPDATE CASCADE,
  distance      INT NOT NULL CHECK (distance >= 0),

  PRIMARY KEY(descendant_id, ancestor_id)
) WITHOUT ROWID;---

CREATE INDEX part_path_idx_ancestor_id         ON part_path(ancestor_id, descendant_id, distance);---
CREATE INDEX part_path_idx_distance_descendant ON part_path(distance, descendant_id, ancestor_id);---
CREATE INDEX part_path_idx_distance_ancestor   ON part_path(distance, ancestor_id, descendant_id);---

-- package_part_split is a view that exists purely to allow for an INSTEAD OF
-- trigger to be used, which automatically splits a package path into parts.
CREATE VIEW package_part_split (
    package_id,
    total_num_parts,
    path_depth,
    part_parent_id,
    part_name,
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

-- insert_package_part_split_on_insert_package is a trigger that fires whenever
-- a new package is inserted.
--
-- This initiates a recursive trigger chain that splits the package's path and
-- inserts them into the part table.
CREATE TRIGGER 
  insert_package_part_split_on_insert_package
AFTER 
  INSERT ON 
    package
BEGIN
  INSERT INTO 
    package_part_split(
      package_id,
      total_num_parts,
      path_depth,
      part_parent_id,
      part_name,
      remaining_path
    )
    SELECT
      new.rowid,
      total_num_parts,
      1,
      NULL,
      substr(remaining_path, 1, slash-1),
      substr(remaining_path, slash+1)
    FROM (
      -- Get the position of the first '/' in the package import path.
      SELECT
        total_num_parts,
        remaining_path,
        instr(remaining_path, '/') AS slash
      FROM (
        -- Get the package import path and append a '/'.
        SELECT
          total_num_parts,
          package_import_path || '/' AS remaining_path
        FROM
          module_package
        WHERE
          package_id = new.rowid
      )
    );
END;---

-- insert_part_insert_part_package_insert_package_part_split_on_insert_package_part_split
-- is a recursive trigger that fires instead of inserting to the
-- package_part_split view.
--
-- It inserts the current part into the part and part_package tables.
--
-- It then inserts the next part, if any, into the package_part_split view.
CREATE TRIGGER insert_part_insert_part_package_insert_package_part_split_on_insert_package_part_split
  INSTEAD OF INSERT ON package_part_split
  BEGIN
    INSERT INTO 
      part(
        name,
        parent_id,
        path_depth,
        package_id
      )
      SELECT
        new.part_name,
        new.part_parent_id,
        new.path_depth,
        iif(new.total_num_parts=new.path_depth, new.package_id, NULL) -- only set the package_id for the final part
      WHERE
        new.part_name != '' -- ignore empty parts, which occurs on the final part
    ON CONFLICT DO
      UPDATE SET
        package_id = excluded.package_id
      WHERE
        excluded.package_id IS NOT NULL; -- only update the package_id if it is not NULL

    INSERT INTO 
      part_package(
        part_id,
        package_id
      )
      SELECT
        new.part_parent_id,
        new.package_id
      WHERE
        new.part_parent_id IS NOT NULL;

    INSERT INTO 
      package_part_split(
        package_id,
        total_num_parts,
        path_depth,
        part_parent_id,
        part_name,
        remaining_path
      )
      SELECT
        new.package_id,
        new.total_num_parts,
        new.path_depth+1,
        part_parent_id,
        substr(new.remaining_path, 1, slash-1),
        substr(new.remaining_path, slash+1)
      FROM (
        -- Get the position of the first '/' in the remaining path and the
        -- rowid of the current part, as the parent id of the next part.
        SELECT
          instr(new.remaining_path, '/') AS slash,
          rowid AS part_parent_id
        FROM
          part
        WHERE
          parent_id IS new.part_parent_id
        AND
          name = new.part_name
      )
    WHERE
      new.path_depth <= new.total_num_parts;
  END;---

-- insert_part_path_on_insert_part is a trigger that fires whenever a new part
-- is inserted into the part table. It populates the part_path table with the
-- new part and all of its ancestors.
CREATE TRIGGER 
  insert_part_path_on_insert_part
AFTER 
  INSERT ON 
    part
BEGIN
  INSERT INTO 
    part_path(
      descendant_id, 
      ancestor_id, 
      distance
    )
  VALUES (
    new.rowid, 
    new.rowid, 
    0
  ) 
  UNION ALL
  SELECT 
    new.rowid, 
    ancestor_id, 
    distance + 1
  FROM 
    part_path
  WHERE 
    descendant_id = new.parent_id;
END;---

-- delete_part_path_on_delete_part is a trigger that fires whenever a part is
-- updated such that its package_id is set to NULL. It deletes the part if it
-- has no children.
CREATE TRIGGER 
  delete_part_on_update_part_package_id_null
AFTER 
  UPDATE OF 
    package_id 
  ON 
    part
  WHEN 
    new.package_id IS NULL
BEGIN
  DELETE FROM
    part
  WHERE
    rowid = new.rowid
  AND
    NOT EXISTS (
      SELECT
        1
      FROM
        part AS p
      WHERE
        p.parent_id IS new.rowid
    );
END;---

-- delete_part_on_delete_part recursively deletes leaf parts that have a NULL
-- package_id.
CREATE TRIGGER 
  delete_part_on_delete_part
AFTER 
  DELETE ON 
    part
BEGIN
  DELETE FROM
    part
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

-- after_update_sync_true_on_main_module fires whenever a module is updated
-- such that sync is set to TRUE. It sets package.keep to FALSE for all of the
-- module's packages. As packages are re-synced, package.keep is set back to
-- TRUE so that only packages that are no longer available in the module are
-- left with keep set to FALSE, resulting in them being deleted to finalize
-- a sync.
CREATE TRIGGER
  after_update_sync_true_on_main_module
AFTER
  UPDATE OF
    sync
  ON 
    main.module
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

