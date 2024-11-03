-- The schema for the database.
--
-- Use ;--- to separate statements. This is a simple hack to allow for
-- splitting complete statements. Omitting the ;--- won't result in an error,
-- but that statement will be executed together with all subsequent statements
-- until the next ;--- or the end of the file.


-- import_path_segment stores all segments for all package import paths as
-- a trie. Root segments have a NULL parent_id.
CREATE TABLE import_path_segment (
  rowid INTEGER PRIMARY KEY,
  parent_id INT NOT NULL
              REFERENCES import_path_segment(rowid)
                ON DELETE RESTRICT
                ON UPDATE CASCADE,
  name      TEXT NOT NULL CHECK (name != ''),
  UNIQUE(parent_id, name),

  path_depth INT NOT NULL CHECK (path_depth > 0),
  package_id INT UNIQUE
                REFERENCES package(rowid)
                  ON DELETE SET NULL
                  ON UPDATE CASCADE
);---

CREATE INDEX import_path_segment_name ON import_path_segment(name);---

-- import_path_segment_closure stores the relationships between all import path
-- segments.
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

CREATE INDEX import_path_segment_closure_descendant_id ON import_path_segment_closure(descendant_id, ancestor_id, distance);---
CREATE INDEX import_path_segment_closure_distance      ON import_path_segment_closure(distance);---

-- import_path_segment_descendant_stats calculates the number of immediate
-- children, number of total descendants, and the maximum descentant distance
-- for each segment.
CREATE VIEW import_path_segment_descendant_stats (
  segment_id,
  num_children,
  num_descendants,
  max_descendant_distance
) AS 
SELECT
  closure.ancestor_id            AS segment_id,
  count(closure.descendant_id) 
    FILTER ( 
      WHERE closure.distance = 1 
    )                            AS num_children,
  count(closure.descendant_id) 
    FILTER ( 
      WHERE closure.distance > 0 
    )                            AS num_descendants,
  max(closure.distance)          AS descendant_max_distance
FROM
  import_path_segment_closure AS closure
GROUP BY
  closure.ancestor_id
;---

-- insert_next_import_path_segment is used by an INSTEAD OF INSERT trigger to
-- recursively split an import path into segments.
CREATE VIEW insert_next_import_path_segment (
  package_id,
  total_num_segments,
  path_depth,
  segment_parent_id,
  segment_name,
  remaining_path
) AS 
VALUES (
  NULL,
  NULL,
  NULL,
  NULL,
  NULL,
  NULL
);---

-- split_import_path_segment_on_insert_package is a trigger that fires whenever
-- a new package is inserted.
--
-- This initiates a recursive trigger chain that splits the package's path and
-- inserts them into the part table.
CREATE TRIGGER 
  split_import_path_segment_on_insert_package
AFTER 
  INSERT ON 
    package
BEGIN

  INSERT INTO insert_next_import_path_segment (
    package_id,
    total_num_segments,
    path_depth,
    segment_parent_id,
    segment_name,
    remaining_path
  )
  SELECT
    package.rowid              AS package_id,
    package.total_num_segments AS total_num_segments,
    0                          AS path_depth,
    NULL                       AS segment_parent_id,
    ''                         AS segment_name,
    package.import_path || '/' AS remaining_path
  FROM
    module_package AS package
  WHERE
    package.rowid = new.rowid;

END;---

-- recursive_package_part_splitter is a recursive trigger that fires instead of
-- inserting to the insert_next_import_path_segment view.
--
-- It inserts the current part into the part and part_package tables.
--
-- It then inserts the next part, if any, into the insert_next_import_path_segment view.
CREATE TRIGGER 
  recursive_package_part_splitter
INSTEAD OF 
  INSERT ON 
    insert_next_import_path_segment
  WHEN
    new.path_depth <= new.total_num_segments
BEGIN

  -- insert the segment
  INSERT INTO import_path_segment (
    parent_id,
    name,
    path_depth,
    package_id
  )
  SELECT
    new.segment_parent_id AS parent_id,
    new.segment_name      AS name,
    new.path_depth        AS path_depth,
    -- the package_id is only set on the final segment, otherwise it is NULL
    iif(
      new.path_depth = new.total_num_segments,
      new.package_id,
      NULL
    )                     AS package_id
  WHERE
    -- the first iteration is invalid, so we skip it
    new.path_depth > 0
  ON CONFLICT DO
    UPDATE SET
      package_id = excluded.package_id
    WHERE
      new.path_depth = new.total_num_segments;

  INSERT INTO
    insert_next_import_path_segment (
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
    new.path_depth + 1     AS path_depth,

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

-- insert_import_path_segment_closure_on_insert_import_path_segment populates
-- the import_path_segment_closure table for each new segment. 
--
-- Each segment is its own ancestor, and all of its parent's ancestors are also
-- its ancestors, with a distance of 1 more than the distance to the parent.
CREATE TRIGGER 
  insert_import_path_segment_closure_on_insert_import_path_segment
AFTER 
  INSERT ON 
    import_path_segment
BEGIN

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
  -- all of the new segment's parent's ancestors, are also its ancestors but
  -- with a distance of 1 more than the distance to the parent
  SELECT 
    closure.ancestor_id  AS ancestor_id, 
    new.rowid            AS descendant_id,
    closure.distance + 1 AS distance
  FROM 
    import_path_segment_closure AS closure
  WHERE
    closure.ancestor_id IS new.parent_id;

END;---

-- delete_import_path_segment_with_null_package_id_and_no_children fires
-- whenever an import path segment's package_id is set to NULL, which occurs
-- automatically when a package is deleted. Segment's must either have
-- children, or be the final segment of a package's import path, otherwise they
-- are deleted. This ensures the trie is kept in sync with the package table.
--
-- Ancestors of a segment with a NULL package_id and no children are cleaned up
-- by the subsequently defined trigger.
CREATE TRIGGER 
  delete_import_path_segment_with_null_package_id_and_no_children
AFTER 
  UPDATE OF 
    package_id 
  ON 
    import_path_segment
  WHEN 
    new.package_id IS NULL
BEGIN

  DELETE FROM
    import_path_segment AS segment
  WHERE
    segment.rowid = new.rowid
  AND
    0 = (
      count(*) FILTER (
        WHERE segment.rowid IS new.rowid
      )
    );

END;---

-- recursively_prune_leaf_parts_with_null_package_id recursively deletes leaf
-- parts that have a NULL package_id.
CREATE TRIGGER 
  recursively_delete_import_path_segment_with_null_package_id_and_no_children
AFTER 
  DELETE ON 
    import_path_segment
BEGIN

  DELETE FROM
    import_path_segment AS segment
  WHERE
    segment.rowid = old.parent_id
  AND
    segment.package_id IS NULL
  AND
    0 = (
      count(*) FILTER (
        WHERE segment.parent_id IS old.parent_id
      )
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
