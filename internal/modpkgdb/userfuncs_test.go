//go:build disable
// +build disable

package modpkgdb

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	RegisterUserFuncs()
})
var _ = Describe("User defined functions", func() {
	var db *DB
	BeforeEach(func(ctx context.Context) {
		dbPath := tempDBPath()
		By("opening db " + dbPath)
		var err error
		db, err = OpenDB(ctx, dbPath)
		Expect(err).
			To(Succeed(), "OpenDB")
		DeferCleanup(func() {
			By("closing the database")
			Expect(db.Close()).
				To(Succeed(), "failed to close database")
		})
	})

	DescribeTable("input", func(ctx context.Context, path, expFirstPart, expRemainingParts string, expNumParts int64) {
		query := fmt.Sprintf("SELECT %s($1), %s($1), %s($1);", FuncFirstPathPart, FuncTrimFirstPathPart, FuncNumPathParts)
		row := db.db.QueryRowContext(ctx, query, path)
		Expect(row.Err()).To(Succeed(), "failed to run query: %q", query)

		var firstPart, remainingParts string
		var numParts int64
		Expect(row.Scan(&firstPart, &remainingParts, &numParts)).To(Succeed(), "failed to scan values")
		Expect(firstPart).To(Equal(expFirstPart), "%s(%q)", FuncFirstPathPart, path)
		Expect(remainingParts).To(Equal(expRemainingParts), "%s(%q)", FuncTrimFirstPathPart, path)
		Expect(numParts).To(Equal(expNumParts), "%s(%q)", FuncNumPathParts, path)
	},
		func(path, _, _ string, _ int64) string { return path },
		Entry(nil, "", "", "", int64(0)),
		Entry(nil, "foo", "foo", "", int64(1)),
		Entry(nil, "bar/foo", "bar", "foo", int64(2)),
		Entry(nil, "baz/bar/foo", "baz", "bar/foo", int64(3)),
	)

})
