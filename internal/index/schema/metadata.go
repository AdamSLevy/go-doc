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
SELECT createdAt, updatedAt, buildRevision, goVersion FROM metadata WHERE rowid=1;
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

func UpsertMetadata(ctx context.Context, db Querier) error {
	const query = `
INSERT INTO metadata(rowid, buildRevision, goVersion) VALUES (1, ?, ?)
  ON CONFLICT(rowid) DO 
    UPDATE SET 
      updatedAt=CURRENT_TIMESTAMP, 
      buildRevision=excluded.buildRevision,
      goVersion=excluded.goVersion;
`
	if _, err := db.ExecContext(ctx, query, buildRevision, goVersion); err != nil {
		return fmt.Errorf("failed to upsert metadata: %w", err)
	}
	return nil
}

var buildRevision, goVersion string = func() (string, string) {
	var buildRevision string
	info, ok := debug.ReadBuildInfo()
	if !ok {
		panic("debug.ReadBuildInfo() failed")
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			buildRevision = s.Value
			break
		}
	}
	return buildRevision, info.GoVersion
}()
