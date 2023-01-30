package index

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenDB(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	ctx := context.Background()

	dbDir := "./"
	// dbDir := t.TempDir()
	dbPath := "file:" + filepath.Join(dbDir, "index.sqlite3")
	db, err := openDB(ctx, dbPath)
	require.NoError(err, "open and initialize")
	require.NoError(db.Close(), "first close")

	db, err = openDB(ctx, dbPath)
	require.NoError(err, "re-open")
	t.Cleanup(func() { require.NoError(db.Close(), "final close") })

	a, err := loadAudit(ctx, db)
	require.NoError(err)
	now := time.Now()
	within := time.Second
	assert.WithinDuration(now, a.CreatedAt, within)
	assert.WithinDuration(now, a.UpdatedAt, within)
	assert.Equal(getBuildRevision(), a.BuildRevision)

	modImportPath := "aslevy.com/go-doc"
	m := _module{
		ImportPath: modImportPath,
		Dir:        "/path/to/" + modImportPath,
		Class:      int(classStdlib),
	}
	moduleId, err := m.upsert(ctx, db)
	require.NoError(err)
	require.Greater(moduleId, int64(0))

	M, err := loadModule(ctx, db, modImportPath)
	require.NoError(err)
	m.ID = moduleId
	require.Equal(m, M)

	pkg1ID, err := insertPackage(ctx, db, moduleId, "")
	require.NoError(err)
	require.Greater(pkg1ID, int64(0))

	P, err := loadPackage(ctx, db, moduleId, "")
	require.NoError(err)
	require.Equal(_package{
		ID:           pkg1ID,
		ModuleID:     moduleId,
		RelativePath: "",
		NumParts:     0,
	}, P)

	pkg2ID, err := insertPackage(ctx, db, moduleId, "internal/index")
	require.NoError(err)
	require.Greater(pkg2ID, int64(0))

	P, err = loadPackage(ctx, db, moduleId, "internal/index")
	require.NoError(err)
	require.Equal(_package{
		ID:           pkg2ID,
		ModuleID:     moduleId,
		RelativePath: "internal/index",
		NumParts:     2,
	}, P)

	require.NoError(insertPartials(ctx, db, pkg1ID, modImportPath))
	require.NoError(insertPartials(ctx, db, pkg2ID, modImportPath+"/internal/index"))

	var count int
	require.NoError(db.QueryRowContext(ctx, "SELECT count(*) FROM module;").Scan(&count))
	require.Equal(1, count, "module")

	require.NoError(db.QueryRowContext(ctx, "SELECT count(*) FROM package;").Scan(&count))
	require.Equal(2, count, "package")

	require.NoError(db.QueryRowContext(ctx, "SELECT count(*) FROM partial;").Scan(&count))
	require.Equal(6, count, "partial")
}
