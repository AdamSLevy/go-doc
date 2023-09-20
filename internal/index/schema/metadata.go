package schema

import (
	"context"
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
	GoSumHash     int32
	Vendor        bool
}

func NewMetadata(mainModPath string) (meta Metadata, err error) {
	meta.BuildRevision, meta.GoVersion, err = parseBuildInfo()
	if err != nil {
		return
	}
	meta.GoSumHash, err = goSumHash(mainModPath)
	if err != nil {
		err = fmt.Errorf("failed to calculate go.sum hash: %w", err)
		return
	}
	meta.Vendor, err = useVendor(mainModPath)
	if err != nil {
		err = fmt.Errorf("failed to determine if vendor is used: %w", err)
		return
	}
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
func goSumHash(mainModPath string) (int32, error) {
	goSumPath := filepath.Join(mainModPath, "go.sum")
	f, err := os.Open(goSumPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	crc := crc32.NewIEEE()
	if _, err := io.Copy(crc, f); err != nil {
		return 0, err
	}
	return int32(crc.Sum32()), nil
}
func useVendor(mainModPath string) (bool, error) {
	vendorPath := filepath.Join(mainModPath, "vendor")
	fi, err := os.Stat(vendorPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !fi.IsDir() {
		return false, fmt.Errorf("vendor path %s is not a directory", vendorPath)
	}
	return true, nil
}

func SelectMetadata(ctx context.Context, db Querier) (Metadata, error) {
	const query = `
SELECT 
  created_at, 
  updated_at, 
  build_revision, 
  go_version,
  go_sum_hash,
  vendor
FROM 
  metadata 
WHERE
  rowid = 1
LIMIT 1;
`
	return scanMetadata(db.QueryRowContext(ctx, query))
}
func scanMetadata(row Scanner) (meta Metadata, _ error) {
	return meta, row.Scan(
		&meta.CreatedAt,
		&meta.UpdatedAt,
		&meta.BuildRevision,
		&meta.GoVersion,
		&meta.GoSumHash,
		&meta.Vendor,
	)
}

func (s *Sync) upsertMetadata(ctx context.Context, meta Metadata) error {
	const query = `
INSERT INTO 
  metadata(
    rowid, 
    build_revision, 
    go_version,
    go_sum_hash,
    vendor
  ) 
VALUES (
  1, ?, ?, ?, ?
)
ON CONFLICT(rowid) DO 
  UPDATE SET (
    updated_at,
    build_revision,
    go_version,
    go_sum_hash,
    vendor
  ) = (
    CURRENT_TIMESTAMP, 
    excluded.build_revision,
    excluded.go_version,
    excluded.go_sum_hash,
    excluded.vendor
  );
`
	_, err := s.tx.ExecContext(ctx, query,
		meta.BuildRevision,
		meta.GoVersion,
		meta.GoSumHash,
		meta.Vendor,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}
	return nil
}
