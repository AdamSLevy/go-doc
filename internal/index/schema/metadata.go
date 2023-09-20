package schema

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

type Metadata struct {
	CreatedAt time.Time
	UpdatedAt time.Time

	BuildRevision string
	GoVersion     string
}

func SelectMetadata(ctx context.Context, db Querier) (Metadata, error) {
	const query = `
SELECT created_at, updated_at, build_revision, go_version FROM metadata WHERE rowid=1;
`
	return scanMetadata(db.QueryRowContext(ctx, query))
}
func scanMetadata(row Scanner) (Metadata, error) {
	var meta Metadata
	return meta, row.Scan(
		&meta.CreatedAt,
		&meta.UpdatedAt,
		&meta.BuildRevision,
		&meta.GoVersion,
	)
}

func (s *Sync) upsertMetadata(ctx context.Context) error {
	const query = `
INSERT INTO 
  metadata(
    rowid, 
    build_revision, 
    go_version
  ) 
VALUES (
  1, ?, ?
)
ON CONFLICT(rowid) DO 
  UPDATE SET 
    updated_at=CURRENT_TIMESTAMP, 
    build_revision=excluded.build_revision,
    go_version=excluded.go_version;
`
	if _, err := s.tx.ExecContext(ctx, query, BuildRevision, GoVersion); err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}
	return nil
}

var BuildRevision, GoVersion string = func() (string, string) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		panic("debug.ReadBuildInfo() failed")
	}
	return parseBuildRevision(info), info.GoVersion
}()

func parseBuildRevision(info *debug.BuildInfo) string {
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return "unknown"
}
