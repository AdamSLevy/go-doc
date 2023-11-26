package db

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"

	"aslevy.com/go-doc/internal/sql"
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
			if s.Value == "" {
				return s.Value
			}
			break
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

func (db *DB) SelectMetadata(ctx context.Context) (*Metadata, error) {
	return selectMetadata(ctx, db.db)
}

//go:embed sql/metadata_select.sql
var querySelectMetadata string

func selectMetadata(ctx context.Context, db sql.Querier) (*Metadata, error) {
	var meta Metadata
	row := db.QueryRowContext(ctx, querySelectMetadata)
	return &meta, row.Scan(
		&meta.CreatedAt,
		&meta.UpdatedAt,
		&meta.BuildRevision,
		&meta.GoVersion,
		&meta.GoModHash,
		&meta.GoSumHash,
		&meta.Vendor,
	)
}

//go:embed sql/metadata_upsert.sql
var queryUpsertMetadata string

func (s *Sync) upsertMetadata(ctx context.Context, meta *Metadata) error {
	_, err := s.tx.ExecContext(ctx, queryUpsertMetadata,
		sql.Named("build_revision", meta.BuildRevision),
		sql.Named("go_version", meta.GoVersion),
		sql.Named("go_mod_hash", meta.GoModHash),
		sql.Named("go_sum_hash", meta.GoSumHash),
		sql.Named("vendor", meta.Vendor),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}
	return nil
}
