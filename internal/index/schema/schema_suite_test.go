package schema_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "aslevy.com/go-doc/internal/index/schema"
)

func TestSchema(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Suite")
}

func initDB(ctx context.Context) *sql.DB {
	GinkgoHelper()
	By("opening the database")
	const (
		dbFile = "file:"
		dbName = "index.sqlite3"
	)
	t := GinkgoT()
	path := dbFile + filepath.Join(t.TempDir(), dbName)
	t.Log("db path: ", path)
	db, err := OpenDB(ctx, path)
	Expect(err).To(Succeed(), "failed to open database")
	Expect(db.PingContext(ctx)).To(Succeed(), "failed to ping database")
	DeferCleanup(func() {
		By("closing the database")
		Expect(db.Close()).To(Succeed(), "failed to close database")
	})
	return db
}

var _ = Describe("Schema", func() {
	var db *sql.DB
	BeforeEach(func(ctx context.Context) {
		db = initDB(ctx)
	})

	BeforeEach(func(ctx context.Context) {
		By("populating the metadata table")
		Expect(UpsertMetadata(ctx, db)).To(Succeed(), "failed to insert metadata")
	})

	Describe("Metadata", func() {
		var md Metadata
		var originalCreatedAt time.Time
		JustBeforeEach(func(ctx context.Context) {
			By("selecting the metadata")
			// Save last loaded created at time for use in next
			// When block
			originalCreatedAt = md.CreatedAt
			var err error
			md, err = SelectMetadata(ctx, db)
			Expect(err).To(Succeed(), "failed to select metadata")
		})

		It("should initialize the metadata", func() {
			Expect(md.CreatedAt).To(BeTemporally("~", time.Now(), time.Second), "CreatedAt should be set to now")
			Expect(md.UpdatedAt).To(Equal(md.CreatedAt), "UpdatedAt should be the same as CreatedAt")
			Expect(md.BuildRevision).ToNot(BeEmpty(), "BuildRevision should be set")
			Expect(md.GoVersion).ToNot(BeEmpty(), "GoVersion should be set")
		})

		When("the metadata already exists", func() {
			BeforeEach(func(ctx context.Context) {
				By("updating the metadata")
				time.Sleep(time.Second)
				Expect(UpsertMetadata(ctx, db)).To(Succeed(), "failed to upsert metadata")
			})
			It("should update the metadata", func() {
				Expect(originalCreatedAt).ToNot(BeZero(), "originalCreatedAt should be set")
				Expect(md.CreatedAt).To(Equal(originalCreatedAt), "CreatedAt should not have changed")
				Expect(md.UpdatedAt).To(BeTemporally(">", md.CreatedAt), "UpdatedAt should be after CreatedAt")
				Expect(md.BuildRevision).ToNot(BeEmpty(), "BuildRevision should be set")
				Expect(md.GoVersion).ToNot(BeEmpty(), "GoVersion should be set")
			})
		})
	})

	var allMods []Module
	BeforeEach(func(ctx context.Context) {
		By("populating the module table")
		allMods = initModules()
		Expect(SyncModules(ctx, db, allMods)).
			To(Equal(allMods), "initial SyncModules should return all modules")
		Expect(SelectAllModules(ctx, db, nil)).
			To(Equal(allMods), "SelectAllModules should return all modules")
	})

	var allPkgs []Package
	BeforeEach(func(ctx context.Context) {
		By("populating the package table")
		allPkgs = initPackages()
		Expect(SyncPackages(ctx, db, allPkgs)).
			To(Equal(allPkgs), "initial SyncPackages should return all packages")
		Expect(SelectAllPackages(ctx, db, nil)).
			To(Equal(allPkgs), "SelectAllPackages should return all packages")
	})

	Describe("Module", func() {
		var newOrChanged []Module
		JustBeforeEach(func(ctx context.Context) {
			By("re-syncing modules")
			var err error
			newOrChanged, err = SyncModules(ctx, db, allMods)
			Expect(err).To(Succeed(), "failed to sync modules")
			Expect(SelectAllModules(ctx, db, nil)).
				To(Equal(allMods), "SelectAllModules should return all modules")
		})
		When("the modules have not changed", func() {
			It("should return no modules", func() {
				Expect(newOrChanged).To(BeEmpty())
			})
		})
		When("a new module is added", func() {
			BeforeEach(func() {
				By("adding a new module")
				allMods = append(allMods, Module{
					ID:         allMods[len(allMods)-1].ID + 1,
					ImportPath: "github.com/onsi/gomega",
					Dir:        "/home/adam/go/pkg/mod/github.com/onsi/gomega@v1.10.3",
					Class:      ClassRequired,
				})
			})
			It("should return the new module", func() {
				Expect(newOrChanged).To(Equal(allMods[len(allMods)-1:]))
			})
		})
		When("a module is removed", func() {
			var removed Module
			BeforeEach(func() {
				By("removing a module")
				removed = allMods[len(allMods)-1]
				allMods = allMods[:len(allMods)-1]
			})
			It("should return nothing", func() {
				Expect(newOrChanged).To(BeEmpty())
			})
			It("should remove the module's packages", func(ctx context.Context) {
				Expect(SelectModulePackages(ctx, db, removed.ID)).
					To(BeEmpty(), "SelectModulePackages should return no packages")
			})
		})
		When("a module is updated", func() {
			BeforeEach(func() {
				By("changing the directory of a module")
				allMods[0].Dir = "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.8.2"
			})
			It("should return the updated module", func() {
				Expect(newOrChanged).To(Equal(allMods[:1]))
			})
		})
		When("modules are removed, added, and updated", func() {
			var removed Module
			BeforeEach(func() {
				By("removing, adding, and updating modules")
				removed = allMods[0]
				allMods = allMods[1:]
				allMods[1].Dir = "/home/adam/go/pkg/mod/github.com/muesli/reflow@v0.3.1"
				allMods = append(allMods, Module{
					ID:         allMods[len(allMods)-1].ID + 1,
					ImportPath: "github.com/onsi/gomega",
					Dir:        "/home/adam/go/pkg/mod/github.com/onsi/gomega@v1.10.3",
					Class:      ClassRequired,
				})
			})
			It("should return the new and updated modules", func() {
				Expect(newOrChanged).To(Equal(allMods[1:]))
			})
			It("should remove the removed module's packages", func(ctx context.Context) {
				Expect(SelectModulePackages(ctx, db, removed.ID)).
					To(BeEmpty(), "SelectModulePackages should return no packages")
			})
		})
	})

	Describe("Package", func() {
		var newOrChanged []Package
		JustBeforeEach(func(ctx context.Context) {
			By("re-syncing packages")
			var err error
			newOrChanged, err = SyncPackages(ctx, db, allPkgs)
			Expect(err).To(Succeed(), "failed to sync packages")
			Expect(SelectAllPackages(ctx, db, nil)).
				To(Equal(allPkgs), "SelectAllPackages should return all packages")
		})
		When("the packages have not changed", func() {
			It("should return no packages", func() {
				Expect(newOrChanged).To(BeEmpty())
			})
		})
		When("a new package is added", func() {
			BeforeEach(func() {
				By("adding a new package")
				allPkgs = append(allPkgs, Package{
					ID:           allPkgs[len(allPkgs)-1].ID + 1,
					ModuleID:     int64(len(allMods)),
					RelativePath: "ginkgo/labels",
					NumParts:     2,
				})
			})
			It("should return the new package", func() {
				Expect(newOrChanged).To(Equal(allPkgs[len(allPkgs)-1:]))
			})
		})
		When("a package is removed", func() {
			BeforeEach(func() {
				By("removing a package")
				allPkgs = allPkgs[:len(allPkgs)-1]
			})
			It("should return nothing", func() {
				Expect(newOrChanged).To(BeEmpty())
			})
		})
	})

	Describe("Part", func() {
		When("a package is removed", func() {
			It("should remove the package's parts not used by any other package", func(ctx context.Context) {
				By("re-syncing packages")
				allPkgs = allPkgs[:len(allPkgs)-2]
				_, err := SyncPackages(ctx, db, allPkgs)
				Expect(err).To(Succeed(), "failed to sync packages")
				Expect(SelectAllPackages(ctx, db, nil)).
					To(Equal(allPkgs), "SelectAllPackages should return all packages")

				By("selecting the parts")
				row := db.QueryRowContext(ctx, `
SELECT count(*) FROM part WHERE name IN ('extensions', 'global', 'table');
`)
				Expect(row.Err()).To(Succeed(), "failed to select parts")
				var count int64
				Expect(row.Scan(&count)).To(Succeed(), "failed to scan count of parts")
				Expect(count).To(BeZero(), "parts should be removed")
			})
		})
		DescribeTable("", func(ctx context.Context, packageID int64, expParts []Part) {
			rows, err := db.QueryContext(ctx, `
SELECT rowid, package_id, name, parent_id FROM part 
        WHERE rowid IN (
                SELECT ancestor_id FROM part_path 
                        WHERE descendant_id = (
                                SELECT rowid FROM part WHERE package_id = ?
                        )
                );
`, packageID)
			Expect(err).To(Succeed(), "failed to select parts")
			defer rows.Close()
			parts := make([]Part, 0, len(expParts))
			for rows.Next() && rows.Err() == nil && ctx.Err() == nil {
				var part Part
				var partPackageID sql.NullInt64
				var partParentID sql.NullInt64
				if !Expect(rows.Scan(
					&part.ID,
					&partPackageID,
					&part.Name,
					&partParentID,
				)).To(Succeed(), "failed to scan part") {
					return
				}
				Expect(rows.Err()).To(Succeed(), "failed to scan parts")
				part.ParentID = -1
				if partParentID.Valid {
					part.ParentID = partParentID.Int64
				}
				part.PackageID = -1
				if partPackageID.Valid {
					part.PackageID = partPackageID.Int64
				}
				parts = append(parts, part)
			}
			Expect(parts).To(Equal(expParts), "parts do not match")
		},
			Entry(nil, int64(1), []Part{{
				ID:        1,
				PackageID: -1,
				ParentID:  -1,
				Name:      "github.com",
			}, {
				ID:        2,
				PackageID: -1,
				ParentID:  1,
				Name:      "stretchr",
			}, {
				ID:        3,
				PackageID: 1,
				ParentID:  2,
				Name:      "testify",
			}}),
			Entry(nil, int64(2), []Part{{
				ID:        1,
				PackageID: -1,
				ParentID:  -1,
				Name:      "github.com",
			}, {
				ID:        2,
				PackageID: -1,
				ParentID:  1,
				Name:      "stretchr",
			}, {
				ID:        3,
				PackageID: 1,
				ParentID:  2,
				Name:      "testify",
			}, {
				ID:        4,
				PackageID: 2,
				ParentID:  3,
				Name:      "assert",
			}}),
			Entry(nil, int64(3), []Part{{
				ID:        1,
				PackageID: -1,
				ParentID:  -1,
				Name:      "github.com",
			}, {
				ID:        2,
				PackageID: -1,
				ParentID:  1,
				Name:      "stretchr",
			}, {
				ID:        3,
				PackageID: 1,
				ParentID:  2,
				Name:      "testify",
			}, {
				ID:        5,
				PackageID: 3,
				ParentID:  3,
				Name:      "require",
			}}),
			Entry(nil, int64(4), []Part{{
				ID:        1,
				PackageID: -1,
				ParentID:  -1,
				Name:      "github.com",
			}, {
				ID:        6,
				PackageID: -1,
				ParentID:  1,
				Name:      "muesli",
			}, {
				ID:        7,
				PackageID: -1,
				ParentID:  6,
				Name:      "reflow",
			}, {
				ID:        8,
				PackageID: 4,
				ParentID:  7,
				Name:      "indent",
			}}),
		)
	})
})

func initModules() []Module {
	return []Module{{
		ID:         1,
		ImportPath: "github.com/stretchr/testify",
		Dir:        "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.8.1",
		Class:      ClassRequired,
	}, {
		ID:         2,
		ImportPath: "github.com/muesli/reflow",
		Dir:        "/home/adam/go/pkg/mod/github.com/muesli/reflow@v0.3.0",
		Class:      ClassRequired,
	}, {
		ID:         3,
		ImportPath: "github.com/onsi/ginkgo/v2",
		Dir:        "/home/adam/go/pkg/mod/github.com/onsi/ginkgo/v2@v2.11.0",
		Class:      ClassRequired,
	}}
}
func initPackages() []Package {
	return []Package{{
		ID:           1,
		ModuleID:     1,
		RelativePath: "",
		NumParts:     0,
	}, {
		ID:           2,
		ModuleID:     1,
		RelativePath: "assert",
		NumParts:     1,
	}, {
		ID:           3,
		ModuleID:     1,
		RelativePath: "require",
		NumParts:     1,
	}, {
		ID:           4,
		ModuleID:     2,
		RelativePath: "indent",
		NumParts:     1,
	}, {
		ID:           5,
		ModuleID:     2,
		RelativePath: "wordwrap",
		NumParts:     1,
	}, {
		ID:           6,
		ModuleID:     2,
		RelativePath: "ansi",
		NumParts:     1,
	}, {
		ID:           7,
		ModuleID:     2,
		RelativePath: "padding",
		NumParts:     1,
	}, {
		ID:           8,
		ModuleID:     3,
		RelativePath: "",
		NumParts:     0,
	}, {
		ID:           9,
		ModuleID:     3,
		RelativePath: "types",
		NumParts:     1,
	}, {
		ID:           10,
		ModuleID:     3,
		RelativePath: "config",
		NumParts:     1,
	}, {
		ID:           11,
		ModuleID:     3,
		RelativePath: "integration",
		NumParts:     1,
	}, {
		ID:           12,
		ModuleID:     3,
		RelativePath: "docs",
		NumParts:     1,
	}, {
		ID:           13,
		ModuleID:     3,
		RelativePath: "extensions/global",
		NumParts:     2,
	}, {
		ID:           14,
		ModuleID:     3,
		RelativePath: "extensions/table",
		NumParts:     2,
	}}
}
