DROP TABLE IF EXISTS temp.module_prune;
CREATE TABLE         temp.module_prune ( 
  rowid INTEGER PRIMARY KEY 
) WITHOUT ROWID;

INSERT INTO temp.module_prune (rowid) 
  SELECT rowid FROM main.module;

DROP TABLE IF EXISTS temp.module_need_sync;
CREATE TABLE         temp.module_need_sync ( 
  rowid INTEGER PRIMARY KEY 
) WITHOUT ROWID;

DROP TABLE IF EXISTS temp.package_prune;
CREATE TABLE         temp.package_prune ( 
  rowid INTEGER PRIMARY KEY 
) WITHOUT ROWID;

DROP TRIGGER IF EXISTS temp.update_module_delete_module_prune;
CREATE TEMP TRIGGER         update_module_delete_module_prune
  AFTER UPDATE ON main.module
  BEGIN
    DELETE FROM module_prune WHERE rowid = new.rowid;
  END;
 
DROP TRIGGER IF EXISTS temp.update_module_insert_module_need_sync_update_package_prune;
CREATE TEMP TRIGGER         update_module_insert_module_need_sync_update_package_prune
  AFTER UPDATE OF dir ON main.module WHEN new.dir != old.dir
  BEGIN
    INSERT INTO module_need_sync (rowid) VALUES (new.rowid);
    INSERT INTO package_prune (rowid) SELECT package.rowid FROM package WHERE module_id = new.rowid;
  END;

DROP TRIGGER IF EXISTS temp.insert_module_insert_module_need_sync;
CREATE TEMP TRIGGER         insert_module_insert_module_need_sync
  AFTER INSERT ON main.module
  BEGIN
    INSERT INTO module_need_sync (rowid) VALUES (new.rowid);
  END;

DROP TRIGGER IF EXISTS temp.update_package_delete_package_prune;
CREATE TEMP TRIGGER         update_package_delete_package_prune
  AFTER UPDATE ON main.package
  BEGIN
    DELETE FROM package_prune WHERE rowid = new.rowid;
  END;

