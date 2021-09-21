package cmd

import (
	"os"
	"path"
	"sort"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Args:  cobra.NoArgs,
	Short: "Recursively clones and updates modules to satisfy all dependencies.",
	Long: `Recursively clones and updates modules to satisfy the dependencies
declared in the MODULE files of each module, starting from the top-level MODULE file.`,
	Run: runSync,
}

var master bool
var update bool
var ignoreErrors bool

func init() {
	// Whether to use 'master' instead of the version specified in the MODULE file.
	syncCmd.Flags().BoolVar(&master, "latest", false, "Use 'origin/master' as the version for all dependencies.")
	syncCmd.Flags().BoolVar(&update, "update", false, "Remove all pinned dependencies.")
	syncCmd.Flags().BoolVar(&ignoreErrors, "ignore-errors", false, "Ignore all errors while pinning and checking dependencies.")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) {
	workspaceRoot := util.GetWorkspaceRoot()
	log.Debug("Workspace: %s.\n", workspaceRoot)

	// Create the DEPS/ subdirectory and create a symlink to the top-level module.
	workspaceModuleName := path.Base(workspaceRoot)
	workspaceModuleSymlink := path.Join(workspaceRoot, util.DepsDirName, workspaceModuleName)
	if !util.DirExists(workspaceModuleSymlink) {
		log.Debug("Creating symlink for the workspace module: '%s/%s' -> '%s'.\n", util.DepsDirName, workspaceModuleName, workspaceRoot)
		util.MkdirAll(path.Dir(workspaceModuleSymlink))
		err := os.Symlink("..", workspaceModuleSymlink)
		if err != nil {
			log.Fatal("Failed to create symlink for workspace module: %s.\n", err)
		}
	}

	workspaceModuleFile := module.ReadModuleFile(workspaceRoot)
	if update || master {
		// Remove all pinned dependencies to start pinning dependencies from scratch.
		workspaceModuleFile.PinnedDependencies = map[string]module.PinnedDependency{}
	}

	// Modules that have been processed.
	done := map[string]bool{}

	// Modules that have been processed.
	fetched := map[string]bool{}

	// Modules that still need to be processed.
	queue := []string{workspaceRoot}

	errorFunc := func(format string, a ...interface{}) {
		log.Error(format, a...)
		log.Fatal("Use --ignore-errors to ignore this error.\n")
	}
	if ignoreErrors {
		errorFunc = log.Warning
	}

	for len(queue) > 0 {
		modulePath := queue[0]
		moduleName := path.Base(modulePath)
		queue = queue[1:]
		if done[modulePath] {
			continue
		}
		done[modulePath] = true

		log.IndentationLevel = 0
		log.Log("Updating %s\n", moduleName)
		log.IndentationLevel = 1

		moduleFile := module.ReadModuleFile(modulePath)

		if len(moduleFile.Dependencies) == 0 {
			log.IndentationLevel = 1
			log.Log("Has no dependencies\n\n")
			continue
		}

		for _, name := range dependencyNames(moduleFile) {
			log.IndentationLevel = 1
			log.Log("Depends on %s\n", name)
			log.IndentationLevel = 2

			dep := moduleFile.Dependencies[name]

			if master {
				dep.Version = masterVersion
			}

			pinnedDep := workspaceModuleFile.PinnedDependencies[name]
			if _, exists := workspaceModuleFile.PinnedDependencies[name]; !exists {
				log.Debug("Dependency pinned to URL '%s', version '%s'.\n", dep.URL, dep.Version)
				pinnedDep.Dependency = dep
			}
			if dep.URL != pinnedDep.URL {
				errorFunc("Dependency requires URL '%s', but URL has been pinned to '%s'.\n", dep.URL, pinnedDep.URL)
			}
			if dep.Version != pinnedDep.Version {
				errorFunc("Dependency requires version '%s', but version has been pinned to '%s'.\n", dep.Version, pinnedDep.Version)
			}

			depModulePath := path.Join(workspaceRoot, util.DepsDirName, name)
			queue = append(queue, depModulePath)

			depModule := module.OpenOrCreateModule(depModulePath, dep.URL)
			if _, exists := fetched[depModulePath]; !exists {
				depModule.Fetch()
				fetched[depModulePath] = true
			}
			if depModule.IsDirty() {
				errorFunc("The exiting module has local changes.\n")
			}
			if depModule.URL() != pinnedDep.URL {
				errorFunc("Dependency requires URL '%s', but the on-disk module has URL '%s'.\n", pinnedDep.URL, depModule.URL())
			}
			if pinnedDep.Hash == "" {
				pinnedDep.Hash = depModule.RevParse(pinnedDep.Version)
			}
			log.Log("Resolved dependency version '%s' to hash '%s...'\n", pinnedDep.Version, pinnedDep.Hash[:10])
			if depModule.Head() != pinnedDep.Hash {
				log.Log("Checking out hash '%s'\n", pinnedDep.Hash[:10])
				depModule.Checkout(pinnedDep.Hash)
				module.SetupModule(depModulePath)
			}

			workspaceModuleFile.PinnedDependencies[name] = pinnedDep
			log.Log("\n")
		}
	}

	module.WriteModuleFile(workspaceRoot, workspaceModuleFile)

	log.IndentationLevel = 0
	log.Success("Done.\n")
}

func dependencyNames(file module.ModuleFile) []string {
	names := []string{}
	for name := range file.Dependencies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
