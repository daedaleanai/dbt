package cmd

import (
	"os"
	"path"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

const masterVersion = "master"

var syncCmd = &cobra.Command{
	Use:   "sync",
	Args:  cobra.NoArgs,
	Short: "Recursively clones and updates modules to satisfy all dependencies.",
	Long: `Recursively clones and updates modules to satisfy the dependencies
declared in the MODULE files of each module, starting from the top-level MODULE file.`,
	Run: runSync,
}

var useMasterVersion bool

func init() {
	// Whether to use 'master' instead of the version specified in the MODULE file.
	syncCmd.Flags().BoolVar(&useMasterVersion, "master", false, "Use 'master' version for all dependencies.")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) {
	workspaceRoot := util.GetWorkspaceRoot()
	log.Log("Current workspace is '%s'.\n", workspaceRoot)

	// The top-level module must already exist and will never be cloned or downloaded by the tool.
	topLevelModule := module.OpenModule(workspaceRoot)
	module.SetupModule(topLevelModule)

	// Create the DEPS/ subdirectory and create a symlink to the top-level module.
	topLevelModuleSymlink := path.Join(workspaceRoot, util.DepsDirName, topLevelModule.Name())
	if !util.DirExists(topLevelModuleSymlink) {
		log.Debug("Creating symlink for the top-level module: '%s/%s' -> '%s'.\n", util.DepsDirName, topLevelModule.Name(), workspaceRoot)

		util.MkdirAll(path.Dir(topLevelModuleSymlink))
		err := os.Symlink("..", topLevelModuleSymlink)
		if err != nil {
			log.Fatal("Failed to create symlink to top-level module: %s.\n", err)
		}
	}

	// Keeps track of the modules whose versions and names have already been fixed and the
	// dependent module that caused the version and name to be fixed.
	fixedModules := map[string]string{topLevelModule.Name(): topLevelModule.Name()}

	// Modules that still need to be processed.
	queue := []module.Module{topLevelModule}

	for len(queue) > 0 {
		mod := queue[0]
		queue = queue[1:]

		log.IndentationLevel = 0
		log.Log("\nProcessing module '%s'.\n", mod.Name())
		log.IndentationLevel = 1

		moduleFile := module.ReadModuleFile(mod.Path())
		changedModuleFile := false
		log.Log("Module has %d dependencies.\n", len(moduleFile.Dependencies))

		for idx, dep := range moduleFile.Dependencies {
			if useMasterVersion {
				dep.Version = module.Version{Rev: masterVersion, Hash: ""}
			}
			log.IndentationLevel = 1
			log.Log("%d) Resolving dependency to module '%s', version '%s' (%s).\n", idx+1, dep.Name, dep.Version.Rev, dep.Version.Hash)
			log.IndentationLevel = 2

			depPath := path.Join(workspaceRoot, util.DepsDirName, dep.Name)
			depMod := module.OpenOrCreateModule(depPath, dep.URL)

			// If the version of the dependency is not yet fixed, the current module will determine
			// the dependency version.
			prevModule, versionIsFixed := fixedModules[dep.Name]
			if !versionIsFixed {
				fixedModules[dep.Name] = mod.Name()
				queue = append(queue, depMod)
			}

			// Check that the module fulfilling the dependency actually comes from URL required
			// by the dependency (i.e., that the dependency module is not a different module that
			// just happens to have the same name).
			if depMod.URL() != dep.URL {
				log.Error("Module URL does not match dependency URL '%s'.\n", dep.URL)
			}

			// If the dependency module has uncommited changes, don't try to change its version.
			if depMod.IsDirty() {
				log.Warning("Module has uncommited changes. Not changing version.\n")
				continue
			}

			if dep.Version.Hash == "" {
				dep.Version.Hash = depMod.RevParse(dep.Version.Rev)
				log.Debug("Dependency version '%s' resolved as hash '%s'.\n", dep.Version.Rev, dep.Version.Hash)
				if !useMasterVersion {
					moduleFile.Dependencies[idx].Version.Hash = dep.Version.Hash
					changedModuleFile = true
				}
			}

			// If the dependency module already has the required hash checked out, there is nothing to do.
			if depMod.Head() == dep.Version.Hash {
				log.Success("Module version hash matches required version.\n")
				continue
			}

			// If the dependency module's version is already fixed, but does not match
			// the version required by the dependency, we issue a warning but do not change the version.
			if versionIsFixed {
				log.Warning("Module version is already fixed by dependent module '%s'. Not changing version.\n", prevModule)
				continue
			}

			log.Log("Checking out version '%s'.\n", dep.Version.Hash)
			depMod.Checkout(dep.Version.Hash)

			// Verify that changing the version has worked.
			if depMod.IsDirty() || depMod.Head() != dep.Version.Hash {
				log.Fatal("Failed to check out required module version '%s'.\n", dep.Version.Hash)
			}
		}

		if changedModuleFile {
			module.WriteModuleFile(mod.Path(), moduleFile)
		}
	}

	log.IndentationLevel = 0
	log.Log("\n")
	log.Success("Done.\n")
}
