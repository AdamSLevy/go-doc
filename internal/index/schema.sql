CREATE TABLE IF NOT EXISTS sync (
  rowid         INTEGER PRIMARY KEY CHECK (rowid = 1),
  createdAt     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updatedAt     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  buildRevision TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS module (
  rowid      INTEGER PRIMARY KEY,
  importPath TEXT UNIQUE,
  dir        TEXT NOT NULL CHECK (dir != ''),
  class      INT  NOT NULL CHECK (class >= 0),
  vendor     BOOL DEFAULT false
);

CREATE INDEX IF NOT EXISTS module_class ON module(class, importPath);

CREATE TABLE IF NOT EXISTS package (
  rowid        INTEGER PRIMARY KEY,
  moduleId     INT  REFERENCES module(rowid) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE,
  relativePath TEXT NOT NULL,
  numParts     INT  GENERATED ALWAYS AS 
    (length(relativePath) - length(replace(relativePath, '/', '')) + -- number of slashes
      iif(length(relativePath)>0,1,0)) -- add 1 if path is not empty
    STORED,

  UNIQUE(moduleId, relativePath) ON CONFLICT IGNORE
);

CREATE TABLE IF NOT EXISTS partial (
  rowid     INTEGER PRIMARY KEY,
  packageId INT  REFERENCES package(rowid) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE,
  parts     TEXT NOT NULL CHECK (parts != ''), -- parts must not be empty
  numParts  INT  GENERATED ALWAYS AS
    (length(parts) - length(replace(parts, '/', '')) + 1) -- number of slashes + 1
    STORED,

  UNIQUE(packageId, parts) ON CONFLICT IGNORE
);

CREATE INDEX IF NOT EXISTS partial_idx_numParts_parts ON partial(numParts, parts COLLATE NOCASE);

CREATE VIEW IF NOT EXISTS packageDir AS
  SELECT 
    package.rowid,
    importPath,
    relativePath,
    dir,
    numParts,
    class,
    vendor 
  FROM package 
    INNER JOIN module 
    ON package.moduleId=module.rowid 
  ORDER BY 
    class ASC, 
    importPath ASC, 
    numParts ASC, 
    relativePath ASC;

CREATE VIEW IF NOT EXISTS partialPackageDir AS
  SELECT
    packageDir.rowid,
    trim((importPath || '/' || relativePath), '/') as packageImportPath,
    trim((dir || '/' || relativePath), '/') as dir,
    importPath as moduleImportPath,
    relativePath,
    packageDir.numParts as packageNumParts,
    parts,
    partial.numParts as partialNumParts
  FROM partial
    INNER JOIN packageDir
    ON partial.packageId=packageDir.rowid
  ORDER BY 
    partial.numParts ASC,
    class ASC, 
    packageDir.importPath ASC, 
    packageDir.numParts ASC, 
    relativePath ASC;
