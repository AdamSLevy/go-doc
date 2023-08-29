package schema

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMetadata(t *testing.T) {
	require := require.New(t)
	db := openTestDB(t)
	ctx := context.Background()

	_, err := SelectMetadata(ctx, db)
	require.ErrorIs(err, sql.ErrNoRows, "expected error")

	require.NoError(UpsertMetadata(ctx, db), "failed to upsert metadata")

	m, err := SelectMetadata(ctx, db)
	require.NoError(err, "failed to select metadata")
	require.WithinDuration(m.CreatedAt, time.Now(), time.Second, "created at time is not within a second of now")
	require.Equal(m.CreatedAt, m.UpdatedAt, "created at and updated at times are not equal")
	require.Equal(BuildRevision, m.BuildRevision, "build revision does not match")
	require.Equal(GoVersion, m.GoVersion, "go version does not match")

	time.Sleep(time.Second)
	require.NoError(UpsertMetadata(ctx, db), "failed to upsert metadata")
	m, err = SelectMetadata(ctx, db)
	require.NoError(err, "failed to select metadata")
	require.Less(m.CreatedAt, m.UpdatedAt, "updated at is not after created at")
}
