CREATE TABLE metadata (
  rowid          INTEGER  PRIMARY KEY CHECK (rowid = 1), -- only one row
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  build_revision  TEXT     NOT NULL CHECK (build_revision != ''),
  go_version      TEXT     NOT NULL CHECK (go_version != '')
);

CREATE TABLE module (
  rowid       INTEGER PRIMARY KEY,
  import_path TEXT    UNIQUE NOT NULL,
  dir         TEXT    NOT NULL CHECK (dir != ''),
  class       INT     NOT NULL CHECK (class >= 0 AND class <= 3), 
  vendor      BOOL    DEFAULT false,
  num_parts   INT     GENERATED ALWAYS AS (
      length(trim(import_path, '/')) 
      - length(replace(trim(import_path, '/'), '/', ''))
      + iif(trim(import_path, '/') = '', 0, 1)
    ) STORED
);

CREATE INDEX module_class ON module(class, import_path);

CREATE TABLE package (
  rowid         INTEGER PRIMARY KEY,
  module_id     INT     REFERENCES module(rowid) 
                          ON DELETE CASCADE 
                          ON UPDATE CASCADE,
  relative_path TEXT    NOT NULL,
  num_parts     INT     GENERATED ALWAYS AS (
      length(trim(relative_path, '/')) 
      - length(replace(trim(relative_path, '/'), '/', ''))
      + iif(trim(relative_path, '/') = '', 0, 1)
    ) STORED,

  UNIQUE(module_id, relative_path) ON CONFLICT IGNORE
);

CREATE VIEW module_package AS
  SELECT 
    package.rowid AS package_id,
    trim(module.import_path || '/' || package.relative_path, '/') as package_import_path,
    rtrim(module.dir        || '/' || package.relative_path, '/') as package_dir,
    package.module_id,
    module.import_path as module_import_path,
    relative_path,
    class, 
    vendor,
    package.num_parts                    as relative_num_parts,
    package.num_parts + module.num_parts as total_num_parts
  FROM package, module
    ON package.module_id=module.rowid 
  ORDER BY 
    class              ASC, 
    module_import_path ASC, 
    relative_num_parts ASC, 
    relative_path      ASC;

CREATE TABLE part (
  rowid      INTEGER PRIMARY KEY,
  parent_id  INT     REFERENCES part(rowid) 
                      ON DELETE CASCADE 
                      ON UPDATE CASCADE,
  name       TEXT    NOT NULL CHECK (name != ''),
  package_id INT REFERENCES package(rowid) 
                   ON DELETE SET NULL
                   ON UPDATE CASCADE,
  path_depth INT NOT NULL CHECK (path_depth > 0),
  UNIQUE(parent_id, name)
);

CREATE        INDEX part_idx_name           ON part(name);
CREATE        INDEX part_idx_package_id     ON part(package_id);
CREATE UNIQUE INDEX part_idx_parent_id_name ON part(parent_id, name) WHERE parent_id IS NOT NULL;
CREATE UNIQUE INDEX part_idx_root_name      ON part(name)            WHERE parent_id IS NULL;

CREATE TABLE part_package (
  part_id    INT REFERENCES part(rowid) 
                   ON DELETE CASCADE 
                   ON UPDATE CASCADE,
  package_id INT REFERENCES package(rowid) 
                   ON DELETE CASCADE 
                   ON UPDATE CASCADE,
  PRIMARY KEY(part_id, package_id)
) WITHOUT ROWID;

CREATE INDEX part_package_idx_package_id ON part_package(package_id);

CREATE TABLE part_path (
  descendant_id INT REFERENCES part(rowid) 
                      ON DELETE CASCADE 
                      ON UPDATE CASCADE,
  ancestor_id   INT REFERENCES part(rowid) 
                      ON DELETE CASCADE 
                      ON UPDATE CASCADE,
  distance      INT NOT NULL CHECK (distance >= 0),

  PRIMARY KEY(descendant_id, ancestor_id)
) WITHOUT ROWID;

CREATE INDEX part_path_idx_ancestor_id         ON part_path(ancestor_id, descendant_id, distance);
CREATE INDEX part_path_idx_descendant_id       ON part_path(descendant_id, ancestor_id, distance);
CREATE INDEX part_path_idx_distance_descendant ON part_path(distance, descendant_id, ancestor_id);
CREATE INDEX part_path_idx_distance_ancestor   ON part_path(distance, ancestor_id, descendant_id);

-- This view exists solely to allow for an INSTEAD OF INSERT trigger to be used
-- to split a package path into parts.
CREATE VIEW package_part_split AS
  SELECT 
    package_id, total_num_parts, 0 AS path_depth, 
    NULL AS part_parent_id, '' AS part_name, package_import_path AS remaining_path FROM
    module_package;

-- This trigger kicks off the recursive trigger to split a package path into
-- parts.
CREATE TRIGGER package_insert_to_package_part_split_insert 
  AFTER INSERT ON package 
  BEGIN
    INSERT INTO package_part_split(
      package_id, total_num_parts, path_depth, 
      part_parent_id, part_name, remaining_path)
      SELECT 
        new.rowid AS package_id, total_num_parts, 1 AS path_depth, 
        NULL AS part_parent_id, substr(remaining_path, 1, slash-1) AS part_name, substr(remaining_path, slash+1) AS remaining_path
      FROM (
        SELECT 
          total_num_parts, remaining_path, instr(remaining_path, '/') AS slash
          FROM (
            SELECT 
              total_num_parts, package_import_path || '/' AS remaining_path
            FROM module_package WHERE package_id = new.rowid
          )
      );
  END;

-- This trigger splits a package path into parts.
CREATE TRIGGER package_part_split_insert_to_part_insert 
  INSTEAD OF INSERT ON package_part_split 
  BEGIN
    INSERT INTO part(name, parent_id, path_depth, package_id)
      SELECT new.part_name, new.part_parent_id, new.path_depth, iif(new.total_num_parts=new.path_depth, new.package_id, NULL)
        WHERE new.part_name != ''
      ON CONFLICT DO 
        UPDATE SET package_id = excluded.package_id 
          WHERE excluded.package_id IS NOT NULL;

    INSERT INTO part_package(part_id, package_id)
      SELECT new.part_parent_id, new.package_id 
        WHERE new.part_parent_id IS NOT NULL;

    INSERT INTO package_part_split(
      package_id, total_num_parts, path_depth, 
      part_parent_id, 
      part_name, remaining_path)
      SELECT 
        new.package_id, new.total_num_parts, new.path_depth+1, 
        (SELECT rowid FROM part WHERE parent_id IS new.part_parent_id AND name = new.part_name) AS part_parent_id,
        substr(new.remaining_path, 1, slash-1), 
        substr(new.remaining_path, slash+1)
      FROM (SELECT instr(new.remaining_path, '/') AS slash)
      WHERE new.path_depth <= new.total_num_parts;
  END;

-- Update part_path table after a part is inserted.
CREATE TRIGGER part_insert_to_part_path_insert 
  AFTER INSERT ON part 
  BEGIN
    INSERT INTO part_path(descendant_id, ancestor_id, distance)
      VALUES (new.rowid, new.rowid, 0) UNION ALL
      SELECT new.rowid, ancestor_id, distance + 1
        FROM part_path
        WHERE descendant_id = new.parent_id;
  END;

-- Remove leaf nodes not referenced by any package.
CREATE TRIGGER part_update_package_id_null_to_part_delete 
  AFTER UPDATE OF package_id ON part 
    WHEN new.package_id IS NULL 
  BEGIN
    DELETE FROM part WHERE 
      package_id IS NULL AND
      rowid NOT IN (SELECT DISTINCT parent_id FROM part WHERE parent_id IS NOT NULL);
  END;

-- Recursively remove leaf nodes not referenced by any package.
CREATE TRIGGER part_delete_to_part_delete 
  AFTER DELETE ON part 
  BEGIN
    DELETE FROM part WHERE 
      package_id IS NULL AND
      rowid NOT IN (SELECT DISTINCT parent_id FROM part WHERE parent_id IS NOT NULL);
  END;
