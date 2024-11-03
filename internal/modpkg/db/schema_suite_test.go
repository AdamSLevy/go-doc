//go:build disable
// +build disable

package db

import (
	"context"
	"path/filepath"
	"testing"

	"aslevy.com/go-doc/internal/dlog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	dlog.Enable()
}

func TestSchema(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schema Suite")
}

func tempDBPath() string {
	GinkgoHelper()
	const dbName = "index.sqlite3"
	tempDir := GinkgoT().TempDir()
	// tempDir = "."
	return filepath.Join(tempDir, dbName)
}

var _ = Describe("Schema", func() {
	var (
		dbPath  string
		db      *DB
		meta    Metadata
		allMods []Module
		allPkgs []Package
	)
	When("OpenDB is called", func() {
		BeforeEach(func(ctx context.Context) {
			dbPath = tempDBPath()
			By("opening db " + dbPath)
			var err error
			db, err = OpenDB(ctx, dbPath)
			Expect(err).
				To(Succeed(), "OpenDB")
			DeferCleanup(func() {
				By("closing the database")
				Expect(db.Close()).
					To(Succeed(), "sql.DB.Close")
			})
		})
		It("initializes the database", func(ctx context.Context) {
			var foreignKeys bool
			Expect(getPragma(ctx, db.db, pragmaForeignKeys, &foreignKeys)).To(Succeed(), "failed to get foreign_keys pragma")
			Expect(foreignKeys).To(BeTrue(), "foreign_keys should be enabled")

			var recursiveTriggers bool
			Expect(getPragma(ctx, db.db, pragmaRecursiveTriggers, &recursiveTriggers)).To(Succeed(), "failed to get recursive_triggers pragma")
			Expect(recursiveTriggers).To(BeTrue(), "recursive_triggers should be enabled")

			applicationID, err := getApplicationID(ctx, db.db)
			Expect(err).To(Succeed(), "failed to get application_id pragma")
			Expect(applicationID).To(Equal(sqliteApplicationID), "application_id should be set")

			userVersion, err := getUserVersion(ctx, db.db)
			Expect(err).To(Succeed(), "failed to get user_version pragma")
			Expect(userVersion).To(Equal(schemaChecksum), "user_version should be set")
		})

		When("first synced", func() {
			BeforeEach(func(ctx context.Context) {
				sync, err := db.StartSync(ctx)
				Expect(err).To(Succeed(), "NewSync")
				Expect(sync).ToNot(BeNil(), "NewSync: should not be nil")

				By("syncing modules")
				allMods = initModules()
				for i := range allMods {
					needsSync, err := sync.AddModule(ctx, &allMods[i])
					Expect(err).To(Succeed(), "Sync.AddModule")
					Expect(needsSync).To(BeTrue(), "Sync.AddModule: all modules should need sync")
				}

				By("syncing packages")
				allPkgs = initPackages()
				for i := range allPkgs {
					Expect(sync.AddPackage(ctx, &allPkgs[i])).
						To(Succeed(), "Sync.AddPackage")
				}

				By("finishing sync")
				meta = initMetadata()
				Expect(sync.Finish(ctx, meta, nil)).
					To(Succeed(), "Sync.Finish")
			})
			It("has all modules and packages", func(ctx context.Context) {
				Expect(SelectAllModules(ctx, db.db)).
					To(Equal(allMods), "all modules should be synced")

				Expect(SelectAllPackages(ctx, db.db)).
					To(Equal(allPkgs), "all packages should be synced")
			})
			When("re-synced", func() {
				var modPrune, needSync []Module
				var pkgPrune []Package
				BeforeEach(func(ctx context.Context) {
					By("closing the database")
					Expect(db.Close()).
						To(Succeed(), "sql.DB.Close")

					By("re-opening the database")
					var err error
					db, err = OpenDB(ctx, dbPath)
					Expect(err).
						To(Succeed(), "OpenDB")

					allMods = initModules()
					allPkgs = initPackages()
					meta = initMetadata()
					modPrune = modPrune[:0]
					needSync = needSync[:0]
					pkgPrune = pkgPrune[:0]
				})
				JustBeforeEach(func(ctx context.Context) {
					var err error
					sync, err := db.StartSync(ctx)
					Expect(err).To(Succeed(), "failed to sync modules")
					Expect(sync).ToNot(BeNil(), "sync should not be nil")

					for i, mod := range allMods {
						syncMod, err := sync.AddModule(ctx, &mod)
						Expect(err).To(Succeed(),
							"failed to add required modules")
						Expect(mod).To(Equal(allMods[i]), "module should not be modified")
						if syncMod {
							needSync = append(needSync, allMods[i])
						}
					}

					if len(needSync) > 0 {
						modIDs := make(map[int64]struct{}, len(needSync))
						for _, mod := range needSync {
							modIDs[mod.ID] = struct{}{}
						}

						By("re-syncing packages")
						for _, pkg := range allPkgs {
							if _, ok := modIDs[pkg.ModuleID]; !ok {
								continue
							}
							Expect(sync.AddPackage(ctx, &pkg)).To(Succeed(),
								"failed to sync packages")
						}
					}

					modPrune, err = sync.selectModulesPrune(ctx)
					Expect(err).To(Succeed(),
						"failed to select modules to prune")

					pkgPrune, err = sync.selectPackagesPrune(ctx)
					Expect(err).To(Succeed(),
						"failed to select packages to prune")

					Expect(sync.Finish(ctx, meta, nil)).To(Succeed(),
						"sync should finish successfully")

					Expect(SelectAllModules(ctx, db.db)).To(Equal(allMods),
						"SelectAllModules should return all modules")

				})

				When("the modules have not changed", func() {
					It("should return no modules", func() {
						Expect(needSync).To(BeEmpty(), "SyncModules should return no modules")
						Expect(modPrune).To(BeEmpty(), "there should be no modules to prune")
						Expect(pkgPrune).To(BeEmpty(), "there should be no packages to prune")
					})
				})

				When("a new module is added", func() {
					BeforeEach(func() {
						By("adding a new module")
						newMod := Module{
							ID:          allMods[len(allMods)-1].ID + 1,
							ImportPath:  "github.com/onsi/gomega",
							RelativeDir: "/home/adam/go/pkg/mod/github.com/onsi/gomega@v1.10.3",
						}
						allMods = append(allMods, newMod)
						allPkgs = append(allPkgs, Package{
							ID:           allPkgs[len(allPkgs)-1].ID + 1,
							RelativePath: "",
							ModuleID:     newMod.ID,
							NumParts:     0,
						}, Package{
							ID:           allPkgs[len(allPkgs)-1].ID + 2,
							RelativePath: "types",
							ModuleID:     newMod.ID,
							NumParts:     1,
						})
					})

					It("should return the new module", func() {
						Expect(needSync).To(Equal(allMods[len(allMods)-1:]), "SyncModules should return the new module")
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
						Expect(needSync).To(BeEmpty(), "SyncModules should return no modules")
						Expect(SelectModulePackages(ctx, db.db, removed[0].ID)).
							To(BeEmpty(), "SelectModulePackages should return no packages")
						Expect(modPrune).To(Equal(removed), "the removed module should be pruned")
						Expect(pkgPrune).To(BeEmpty(), "there should be no packages to prune")
					})
				})

				When("a module is updated", func() {
					var updated []Module
					var modPkgs []Package
					BeforeEach(func(ctx context.Context) {
						By("changing the directory of a module")
						allMods[0].RelativeDir = "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.8.2"
						updated = allMods[:1]

						var err error
						modPkgs, err = SelectModulePackages(ctx, db.db, updated[0].ID)
						Expect(err).To(Succeed(), "failed to select module packages")
						Expect(modPkgs).ToNot(BeEmpty(), "module packages should not be empty")
					})

					It("should return the updated module", func() {
						Expect(needSync).To(Equal(updated), "SyncModules should return the updated module")
						Expect(modPrune).To(BeEmpty(), "there should be no modules to prune")
						Expect(pkgPrune).To(BeEmpty(), "the module's packages should be potentially pruned")
					})

					When("the module's packages are unchanged", func() {
						It("should retain the module's packages", func(ctx context.Context) {
							Expect(SelectModulePackages(ctx, db.db, updated[0].ID)).
								To(Equal(modPkgs), "synced packages are not correct")
						})
					})

					When("a module's package is removed", func() {
						BeforeEach(func() {
							By("removing a package")
							allPkgs = modPkgs[1:]
						})
						It("should prune the removed package", func(ctx context.Context) {
							Expect(SelectModulePackages(ctx, db.db, updated[0].ID)).
								To(Equal(allPkgs), "synced packages are not correct")
						})
					})

					When("a module's packages are added and removed", func() {
						BeforeEach(func() {
							modPkgs = append(modPkgs, Package{
								ID:           int64(len(allPkgs) + 1),
								ModuleID:     updated[0].ID,
								RelativePath: "added",
								NumParts:     1,
							})
							allPkgs = modPkgs[1:]
						})

						It("should prune the removed package, add the added package, and retain the pre-existing packages", func(ctx context.Context) {
							Expect(SelectModulePackages(ctx, db.db, updated[0].ID)).
								To(Equal(allPkgs), "synced packages are not correct")
						})
					})
				})

				When("modules are removed, added, and updated", func() {
					var removed Module
					BeforeEach(func() {
						By("removing, adding, and updating modules")
						removed = allMods[0]
						allMods = allMods[1:]
						allMods[1].RelativeDir = "/home/adam/go/pkg/mod/github.com/muesli/reflow@v0.3.1"
						newMod := Module{
							ID:          allMods[len(allMods)-1].ID + 1,
							ImportPath:  "github.com/onsi/gomega",
							RelativeDir: "/home/adam/go/pkg/mod/github.com/onsi/gomega@v1.10.3",
						}
						allMods = append(allMods, newMod)
						allPkgs = append(allPkgs, Package{
							ID:           allPkgs[len(allPkgs)-1].ID + 1,
							RelativePath: "",
							ModuleID:     newMod.ID,
							NumParts:     0,
						}, Package{
							ID:           allPkgs[len(allPkgs)-1].ID + 2,
							RelativePath: "types",
							ModuleID:     newMod.ID,
							NumParts:     1,
						})
					})
					It("should return the new and updated modules", func() {
						Expect(needSync).To(Equal(allMods[1:]))
					})
					It("should remove the removed module's packages", func(ctx context.Context) {
						Expect(SelectModulePackages(ctx, db.db, removed.ID)).
							To(BeEmpty(), "SelectModulePackages should return no packages")
					})
				})
			})
		})
	})

	// Describe("Metadata", func() {
	// 	var md Metadata
	// 	var originalCreatedAt time.Time
	// 	JustBeforeEach(func(ctx context.Context) {
	// 		By("selecting the metadata")
	// 		// Save last loaded created at time for use in next
	// 		// When block
	// 		originalCreatedAt = md.CreatedAt
	// 		var err error
	// 		md, err = SelectMetadata(ctx, db)
	// 		Expect(err).To(Succeed(), "failed to select metadata")
	// 	})

	// 	It("should initialize the metadata", func() {
	// 		Expect(md.CreatedAt).To(BeTemporally("~", time.Now(), time.Second), "CreatedAt should be set to now")
	// 		Expect(md.UpdatedAt).To(Equal(md.CreatedAt), "UpdatedAt should be the same as CreatedAt")
	// 		Expect(md.BuildRevision).ToNot(BeEmpty(), "BuildRevision should be set")
	// 		Expect(md.GoVersion).ToNot(BeEmpty(), "GoVersion should be set")
	// 	})

	// 	When("the metadata already exists", func() {
	// 		BeforeEach(func(ctx context.Context) {
	// 			By("updating the metadata")
	// 			time.Sleep(time.Second)
	// 			Expect(UpsertMetadata(ctx, db)).To(Succeed(), "failed to upsert metadata")
	// 		})

	// 		It("should update the metadata", func() {
	// 			Expect(originalCreatedAt).ToNot(BeZero(), "originalCreatedAt should be set")
	// 			Expect(md.CreatedAt).To(Equal(originalCreatedAt), "CreatedAt should not have changed")
	// 			Expect(md.UpdatedAt).To(BeTemporally(">", md.CreatedAt), "UpdatedAt should be after CreatedAt")
	// 			Expect(md.BuildRevision).ToNot(BeEmpty(), "BuildRevision should be set")
	// 			Expect(md.GoVersion).ToNot(BeEmpty(), "GoVersion should be set")
	// 		})
	// 	})
	// })

})

func initMetadata() Metadata {
	return Metadata{
		BuildRevision: "test",
		GoVersion:     "test",
		GoModHash:     1024,
		GoSumHash:     1024,
	}
}

func initModules() []Module {
	return []Module{{
		ID:          1,
		ImportPath:  "github.com/stretchr/testify",
		RelativeDir: "/home/adam/go/pkg/mod/github.com/stretchr/testify@v1.8.1",
	}, {
		ID:          2,
		ImportPath:  "github.com/muesli/reflow",
		RelativeDir: "/home/adam/go/pkg/mod/github.com/muesli/reflow@v0.3.0",
	}, {
		ID:          3,
		ImportPath:  "github.com/onsi/ginkgo/v2",
		RelativeDir: "/home/adam/go/pkg/mod/github.com/onsi/ginkgo/v2@v2.11.0",
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
