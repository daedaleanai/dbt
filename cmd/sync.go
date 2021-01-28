package cmd

import (
	"dbt/log"
	"dbt/module"
	"dbt/util"
	"os"
	"path"

	"github.com/spf13/cobra"
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
	rootModule := module.OpenModule(workspaceRoot)
	module.SetupModule(rootModule)

	// Create the DEPS/ subdirectory and create a symlink to the top-level module.
	rootModuleSymlink := path.Join(workspaceRoot, util.DepsDirName, rootModule.Name())
	if !util.DirExists(rootModuleSymlink) {
		log.Debug("Creating symlink for the top-level module: '%s/%s' -> '%s'.\n", util.DepsDirName, rootModule.Name(), workspaceRoot)

		err := os.MkdirAll(path.Dir(rootModuleSymlink), util.DirMode)
		if err != nil {
			log.Error("Failed to create %s/ directory: %s.\n", util.DepsDirName, err)
		}
		err = os.Symlink("..", rootModuleSymlink)
		if err != nil {
			log.Error("Failed to create symlink to top-level module: %s.\n", err)
		}
	}

	// Keeps track of the modules whose versions have already been fixed and the
	// dependent module that caused the version to be fixed.
	fixed := map[string]string{rootModule.Name(): rootModule.Name()}

	// Modules that still need to be processed.
	queue := []module.Module{rootModule}

	for len(queue) > 0 {
		mod := queue[0]
		queue = queue[1:]

		log.IndentationLevel = 0
		log.Log("\nProcessing module '%s'.\n", mod.Name())
		log.IndentationLevel = 1

		deps := module.ReadModuleFile(mod.Path())
		log.Log("Module has %d dependencies.\n", len(deps))

		for idx, dep := range deps {
			if useMasterVersion {
				dep.Version = masterVersion
			}
			log.IndentationLevel = 1
			log.Log("%d) Resolving dependency to '%s', version '%s'.\n", idx+1, dep.ModuleName(), dep.Version)
			log.IndentationLevel = 2

			depPath := path.Join(workspaceRoot, util.DepsDirName, dep.ModuleName())
			depMod := module.OpenOrCreateModule(depPath, dep.URL)

			// If the version of the dependency is not yet fixed, the current module will determine
			// the dependency version.
			dependentModule, versionIsFixed := fixed[depMod.Name()]
			if !versionIsFixed {
				fixed[depMod.Name()] = mod.Name()
				queue = append(queue, depMod)
			}

			// Check that the module fulfilling dependency actually comes from source required
			// by the dependency. I.e., that the dependency module is not a different module that
			// just happens to have the same name.
			if !depMod.HasOrigin(dep.URL) {
				log.Warning("Module origin does not match dependency URL '%s'.\n", dep.URL)
			}

			// If the dependency module has uncommited changes, don't try to change its version.
			if depMod.IsDirty() {
				log.Warning("Module is in a dirty state. Not changing version.\n")
				continue
			}

			// If the dependency module already has the required version checked out, there is nothing to do.
			if depMod.HasVersionCheckedOut(dep.Version) {
				log.Success("Module version matches required version.\n")
				continue
			}

			// If the dependency module's version is already fixed, but does not match
			// the version required by the dependency, we issue a warning but do not change the version.
			if versionIsFixed {
				log.Warning("Module version is already fixed by dependent module '%s'. Not changing version.\n", dependentModule)
				continue
			}

			log.Log("Changing version to '%s'.\n", dep.Version)
			depMod.CheckoutVersion(dep.Version)

			// Verify that changing the version has worked.
			if !depMod.HasVersionCheckedOut(dep.Version) {
				log.Error("Failed to check out required module version '%s'.\n", dep.Version)
			}
		}
	}

	log.IndentationLevel = 0
	log.Log("\n")
	log.Success("Done.\n")
}
