package db

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

type Metadata struct {
	CreatedAt time.Time
	UpdatedAt time.Time

	BuildRevision string
	GoVersion     string

	GoModHash int32
	GoSumHash int32

	Vendor bool
}

func NewMetadata(mainModuleDir string) (meta Metadata, rerr error) {
	var err error
	meta.BuildRevision, meta.GoVersion, err = parseBuildInfo()
	rerr = errors.Join(rerr, err)

	meta.GoModHash, err = hashGoModFile(mainModuleDir)
	rerr = errors.Join(rerr, err)

	meta.GoSumHash, err = hashGoSumFile(mainModuleDir)
	rerr = errors.Join(rerr, err)

	meta.Vendor, err = usingVendor(mainModuleDir)
	rerr = errors.Join(rerr, err)

	return
}
func parseBuildInfo() (string, string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", "", fmt.Errorf("debug.ReadBuildInfo() failed")
	}
	return parseBuildRevision(info), info.GoVersion, nil
}
func parseBuildRevision(info *debug.BuildInfo) string {
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return "unknown"
}

func hashGoModFile(mainModDir string) (int32, error) {
	return fileCRC32(filepath.Join(mainModDir, "go.mod"))
}
func hashGoSumFile(mainModDir string) (int32, error) {
	return fileCRC32(filepath.Join(mainModDir, "go.sum"))
}
func fileCRC32(filePath string) (int32, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open %q: %w", filePath, err)
	}
	defer f.Close()

	crc := crc32.NewIEEE()
	if _, err := io.Copy(crc, f); err != nil {
		return 0, fmt.Errorf("failed to write file %q to CRC32 hash: %w", filePath, err)
	}
	return int32(crc.Sum32()), nil
}
func usingVendor(mainModDir string) (bool, error) {
	vendorPath := filepath.Join(mainModDir, "vendor")
	fi, err := os.Stat(vendorPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat %q: %w", vendorPath, err)
	}
	if !fi.IsDir() {
		return false, fmt.Errorf("vendor path %q is not a directory", vendorPath)
	}
	return true, nil
}

func (db *DB) SelectMetadata(ctx context.Context) (Metadata, error) {
	const query = `
SELECT 
  created_at, 
  updated_at, 

  build_revision, 
  go_version,

  go_mod_hash,
  go_sum_hash,
  vendor
FROM 
  metadata 
WHERE
  rowid = 1
LIMIT 1;
`
	return scanMetadata(db.db.QueryRowContext(ctx, query))
}
func scanMetadata(row Scanner) (meta Metadata, _ error) {
	return meta, row.Scan(
		&meta.CreatedAt,
		&meta.UpdatedAt,

		&meta.BuildRevision,
		&meta.GoVersion,

		&meta.GoModHash,
		&meta.GoSumHash,
		&meta.Vendor,
	)
}

func (s *Sync) upsertMetadata(ctx context.Context, meta *Metadata) error {
	const query = `
INSERT INTO 
  metadata(
    rowid, 

    build_revision, 
    go_version,

    go_mod_hash,
    go_sum_hash,
    vendor
  ) 
VALUES (
  1, 
  ?, ?, ?, ?,
  ?, ?, ?, ?
)
ON CONFLICT(rowid) DO 
  UPDATE SET 
    updated_at = CURRENT_TIMESTAMP, 
    (
      build_revision,
      go_version,
 
      go_mod_hash,
      go_sum_hash,
      vendor
    ) = (
      excluded.build_revision,
      excluded.go_version,

      excluded.go_mod_hash,
      excluded.go_sum_hash,
      excluded.vendor
    );
`
	_, err := s.tx.ExecContext(ctx, query,
		meta.BuildRevision,
		meta.GoVersion,

		meta.GoModHash,
		meta.GoSumHash,
		meta.Vendor,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}
	return nil
}
