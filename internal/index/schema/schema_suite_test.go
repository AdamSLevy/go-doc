package schema

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// func init() {
// 	dlog.Enable()
// }

func TestSchema(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Suite")
}

func tempDBPath() string {
	GinkgoHelper()
	const (
		dbFile = "file:"
		dbName = "index.sqlite3"
	)
	tempDir := GinkgoT().TempDir()
	// tempDir = "."
	return dbFile + filepath.Join(tempDir, dbName)
}
func initDB(ctx context.Context, dbPath string) *sql.DB {
	GinkgoHelper()
	By("opening db " + dbPath)
	db, err := OpenDB(ctx, dbPath)
	Expect(err).To(Succeed(), "failed to open database")
	DeferCleanup(func() {
		By("closing the database")
		Expect(db.Close()).To(Succeed(), "failed to close database")
	})

	applicationID, err := getApplicationID(ctx, db)
	Expect(err).To(Succeed(), "failed to get application_id pragma")
	Expect(applicationID).To(Equal(sqliteApplicationID), "application_id should be set")

	userVersion, err := getUserVersion(ctx, db)
	Expect(err).To(Succeed(), "failed to get user_version pragma")
	Expect(userVersion).To(Equal(schemaCRC), "user_version should be set")

	var foreignKeys, recursiveTriggers bool
	Expect(getPragma(ctx, db, pragmaForeignKeys, &foreignKeys)).To(Succeed(), "failed to get foreign_keys pragma")
	Expect(foreignKeys).To(BeTrue(), "foreign_keys should be enabled")

	Expect(getPragma(ctx, db, pragmaRecursiveTriggers, &recursiveTriggers)).To(Succeed(), "failed to get recursive_triggers pragma")
	Expect(recursiveTriggers).To(BeTrue(), "recursive_triggers should be enabled")

	return db
}

var _ = Describe("Schema", func() {
	var (
		db      *sql.DB
		allMods []Module
		allPkgs []Package
	)
	BeforeEach(func(ctx context.Context) {
		dbPath := tempDBPath()
		db = initDB(ctx, dbPath)

		By("populating the metadata table")
		Expect(UpsertMetadata(ctx, db)).To(Succeed(), "failed to insert metadata")

		By("populating the module table")
		allMods = initModules()
		Expect(SyncModules(ctx, db, allMods)).
			To(Equal(allMods), "initial SyncModules should return all modules")
		Expect(SelectAllModules(ctx, db, nil)).
			To(Equal(allMods), "SelectAllModules should return all modules")

		By("populating the package table")
		allPkgs = initPackages()
		Expect(SyncPackages(ctx, db, allPkgs)).
			To(Succeed(), "initial SyncPackages should succeed")
		Expect(SelectAllPackages(ctx, db, nil)).
			To(Equal(allPkgs), "SelectAllPackages should return all packages")

		By("closing and re-opening the database")
		Expect(db.Close()).To(Succeed(), "failed to close database")
		db = initDB(ctx, dbPath)
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

	Describe("Sync", func() {
		var newOrChanged, modPrune, needSync []Module
		var pkgPrune []Package
		JustBeforeEach(func(ctx context.Context) {
			By("re-syncing modules")
			var err error
			newOrChanged, err = SyncModules(ctx, db, allMods)
			Expect(err).To(Succeed(), "failed to sync modules")
			Expect(SelectAllModules(ctx, db, nil)).
				To(Equal(allMods), "SelectAllModules should return all modules")

			modPrune, err = selectModulesPrune(ctx, db, nil)
			Expect(err).To(Succeed(), "failed to select modules to prune")

			needSync, err = selectModulesNeedSync(ctx, db, nil)
			Expect(err).To(Succeed(), "failed to select modules to sync")
			Expect(needSync).To(Equal(newOrChanged), "modules to sync should be equal to new or changed modules")

			pkgPrune, err = selectPackagesPrune(ctx, db, nil)
			Expect(err).To(Succeed(), "failed to select packages to prune")
		})

		When("the modules have not changed", func() {
			It("should return no modules", func() {
				Expect(newOrChanged).To(BeEmpty(), "SyncModules should return no modules")
				Expect(modPrune).To(BeEmpty(), "there should be no modules to prune")
				Expect(pkgPrune).To(BeEmpty(), "there should be no packages to prune")
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
				Expect(newOrChanged).To(Equal(allMods[len(allMods)-1:]), "SyncModules should return the new module")
				Expect(modPrune).To(BeEmpty(), "there should be no modules to prune")
				Expect(pkgPrune).To(BeEmpty(), "there should be no packages to prune")
			})
		})

		When("a module is removed", func() {
			var removed []Module
			BeforeEach(func() {
				By("removing a module")
				removed = allMods[len(allMods)-1:]
				allMods = allMods[:len(allMods)-1]
			})

			It("should prune the removed module and its packages", func(ctx context.Context) {
				Expect(newOrChanged).To(BeEmpty(), "SyncModules should return no modules")
				Expect(SelectModulePackages(ctx, db, removed[0].ID)).
					To(BeEmpty(), "SelectModulePackages should return no packages")
				for i, mod := range removed {
					removed[i] = Module{ID: mod.ID}
				}
				Expect(modPrune).To(Equal(removed), "the removed module should be pruned")
				Expect(pkgPrune).To(BeEmpty(), "there should be no packages to prune")
			})
		})

		When("a module is updated", func() {
			var updated []Module
			var modPkgs []Package
			BeforeEach(func(ctx context.Context) {
				By("changing the directory of a module")
				allMods[0].Dir = "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.8.2"
				updated = allMods[:1]

				var err error
				modPkgs, err = SelectModulePackages(ctx, db, updated[0].ID)
				Expect(err).To(Succeed(), "failed to select module packages")
				Expect(modPkgs).ToNot(BeEmpty(), "module packages should not be empty")
			})

			It("should return the updated module", func() {
				Expect(newOrChanged).To(Equal(updated), "SyncModules should return the updated module")
				Expect(modPrune).To(BeEmpty(), "there should be no modules to prune")
				Expect(pkgPrune).To(Equal(modPkgs), "the module's packages should be potentially pruned")
			})

			DescribeTable("SyncPackages", func(ctx context.Context, start, endOffset int,
				added ...Package) {
				By("re-syncing packages")
				modPkgs = append(modPkgs[start:len(modPkgs)+endOffset], added...)
				Expect(SyncPackages(ctx, db, modPkgs)).
					To(Succeed(), "failed to sync packages")

				Expect(SelectModulePackages(ctx, db, updated[0].ID)).
					To(Equal(modPkgs), "synced packages are not correct")
			},
				Entry("packages unchanged", 0, 0),
				Entry("package removed", 0, -1),
				Entry("package added", 0, 0, Package{
					ID:           15,
					ModuleID:     1,
					RelativePath: "added",
					NumParts:     1,
				}),
				Entry("package added and removed", 0, -1, Package{
					ID:           15,
					ModuleID:     1,
					RelativePath: "added",
					NumParts:     1,
				}),
			)

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
