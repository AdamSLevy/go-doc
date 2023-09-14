-- These statements are run at the beginning of a sync.
--
-- They create temporary tables and triggers that are used to track which
-- modules need to be re-synced or deleted, and which packages need to be
-- deleted.

-- module_prune tracks modules that may need to be deleted after the sync.
CREATE TEMP TABLE module_prune (
  rowid INTEGER PRIMARY KEY
) WITHOUT ROWID;---

-- We begin a sync by inserting all modules into module_prune.
INSERT INTO temp.module_prune (rowid)
  SELECT rowid FROM main.module;---

-- delete_module_prune_on_update_module deletes modules from module_prune when
-- they are updated.
--
-- During sync we upsert all required modules using an ON CONFLICT clause which
-- ensures this update trigger fires even if the module is not changed.
CREATE TEMP TRIGGER delete_module_prune_on_update_module
  AFTER UPDATE ON main.module
  BEGIN
    DELETE FROM module_prune WHERE rowid = new.rowid;
  END;---

-- module_need_sync tracks modules which need their packages to be resynced
-- because they are new or have changed.
CREATE TEMP TABLE module_need_sync (
  rowid INTEGER PRIMARY KEY
) WITHOUT ROWID;---

-- package_prune tracks packages that may need to be deleted after the sync.
CREATE TEMP TABLE package_prune (
  rowid INTEGER PRIMARY KEY
) WITHOUT ROWID;---

-- insert_module_need_sync_insert_package_prune_on_update_module_dir inserts
-- modules whose dir has changed and thus need to be re-synced into
-- module_need_sync. It also inserts all of that module's packages into
-- package_prune.
CREATE TEMP TRIGGER insert_module_need_sync_insert_package_prune_on_update_module_dir
  AFTER UPDATE OF dir ON main.module WHEN new.dir != old.dir
  BEGIN
    INSERT INTO module_need_sync (rowid)
      VALUES (new.rowid);
    INSERT INTO package_prune (rowid)
      SELECT package.rowid
      FROM package
      WHERE module_id = new.rowid;
  END;---

-- insert_module_need_sync_on_insert_module inserts new modules into
-- module_need_sync.
CREATE TEMP TRIGGER insert_module_need_sync_on_insert_module
  AFTER INSERT ON main.module
  BEGIN
    INSERT INTO module_need_sync (rowid) VALUES (new.rowid);
  END;---

-- delete_package_prune_on_update_package deletes packages from package_prune
-- when they are updated.
--
-- During sync we upsert all packages for all modules where are new or changed
-- using an ON CONFLICT clause which ensures this trigger fires even if the
-- package is not changed.
CREATE TEMP TRIGGER delete_package_prune_on_update_package
  AFTER UPDATE ON main.package
  BEGIN
    DELETE FROM package_prune WHERE rowid = new.rowid;
  END;---
