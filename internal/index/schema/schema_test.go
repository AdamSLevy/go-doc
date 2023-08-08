package schema

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func openTestDB(t testing.TB) *sql.DB {
	require := require.New(t)
	const (
		dbFile = "file:"
		dbName = "index.sqlite3"
	)
	path := dbFile + filepath.Join(t.TempDir(), dbName)
	t.Log("db path: ", path)

	db, err := OpenDB(context.Background(), path)
	require.NoError(err, "failed to open database")
	t.Cleanup(func() { require.NoError(db.Close(), "failed to close database") })
	return db
}
