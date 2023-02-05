CREATE TABLE IF NOT EXISTS sync (
  rowid         INTEGER  PRIMARY KEY CHECK (rowid = 1), -- only one row
  createdAt     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updatedAt     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  buildRevision TEXT     NOT NULL
);

CREATE TABLE IF NOT EXISTS module (
  rowid      INTEGER PRIMARY KEY,
  importPath TEXT    UNIQUE NOT NULL,
  dir        TEXT    NOT NULL CHECK (dir != ''), -- dir must not be empty
  class      INT     NOT NULL CHECK (class >= 0 AND class <= 3), -- 0: stdlib, 1: local, 2: required, 3: not required
  vendor     BOOL    DEFAULT false,
  numParts   INT     GENERATED ALWAYS AS 
                       (length(importPath) - length(replace(importPath, '/', '')) + -- number of slashes
                         iif(length(importPath)>0,1,0)) -- add 1 if path is not empty
                       STORED
);
CREATE INDEX IF NOT EXISTS module_class ON module(class, importPath);

CREATE TABLE IF NOT EXISTS package (
  rowid        INTEGER PRIMARY KEY,
  moduleId     INT     REFERENCES module(rowid) 
                         ON DELETE CASCADE 
                         ON UPDATE CASCADE,
  relativePath TEXT    NOT NULL,
  numParts     INT     GENERATED ALWAYS AS 
                         (length(relativePath) - length(replace(relativePath, '/', '')) + -- number of slashes
                           iif(length(relativePath)>0,1,0)) -- add 1 if path is not empty
                         STORED,

  UNIQUE(moduleId, relativePath) ON CONFLICT IGNORE
);

CREATE VIEW IF NOT EXISTS modulePackage AS
  SELECT 
    package.rowid,
    trim(module.importPath || '/' || package.relativePath, '/') as packageImportPath,
    rtrim(module.dir        || '/' || package.relativePath, '/') as packageDir,
    package.moduleId,
    module.importPath as moduleImportPath,
    relativePath,
    class, 
    vendor,
    package.numParts                   as relativeNumParts,
    package.numParts + module.numParts as totalNumParts
  FROM package 
    INNER JOIN module
    ON package.moduleId=module.rowid 
  ORDER BY 
    class            ASC, 
    moduleImportPath ASC, 
    relativeNumParts ASC, 
    relativePath     ASC;

CREATE TABLE IF NOT EXISTS partial (
  rowid     INTEGER PRIMARY KEY,
  packageId INT     REFERENCES package(rowid) 
                      ON DELETE CASCADE 
                      ON UPDATE CASCADE,
  parts     TEXT    NOT NULL CHECK (parts != ''), -- parts must not be empty
  numParts  INT     GENERATED ALWAYS AS
                      (length(parts) - length(replace(parts, '/', '')) + 1) -- number of slashes + 1
                      STORED,

  UNIQUE(packageId, parts) ON CONFLICT IGNORE
);

CREATE INDEX IF NOT EXISTS partial_idx_numParts_parts ON partial(numParts, parts COLLATE NOCASE);

CREATE VIEW IF NOT EXISTS partialPackage AS
  SELECT
    package.rowid,
    packageImportPath,
    packageDir,
    moduleId,
    moduleImportPath,
    relativePath,
    class,
    vendor,
    relativeNumParts,
    totalNumParts,
    parts,
    partial.numParts as partialNumParts
  FROM partial
    INNER JOIN modulePackage AS package
    ON partial.packageId=package.rowid
  ORDER BY 
    partialNumParts  ASC,
    class            ASC, 
    moduleImportPath ASC,
    relativeNumParts ASC,
    relativePath     ASC;
