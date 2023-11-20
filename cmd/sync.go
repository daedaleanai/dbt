package cmd

import (
	"errors"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/daedaleanai/dbt/v3/log"
	"github.com/daedaleanai/dbt/v3/module"
	"github.com/daedaleanai/dbt/v3/util"

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

var update bool
var ignoreErrors bool
var strict bool

func init() {
	// Whether to use 'master' instead of the version specified in the MODULE file.
	syncCmd.Flags().BoolVar(&update, "update", false, "Recompute all dependency hashes based on the version string.")
	syncCmd.Flags().BoolVar(&ignoreErrors, "ignore-errors", false, "Ignore all errors while pinning and checking dependencies.")
	syncCmd.Flags().BoolVar(&strict, "strict", false, "Check that all dependency hashes are present and the chosen commit is an ancestor of the commit described by version string.")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) {
	if update && strict {
		log.Fatal("--update and --strict can not be used together.\n")
	}

	workspaceRoot := util.GetWorkspaceRoot()
	log.Debug("Workspace: %s.\n", workspaceRoot)

	workspaceModuleFile := module.ReadModuleFile(workspaceRoot)
	workspaceModuleName := module.OpenModule(workspaceRoot).Name()
	log.Debug("Workspace module name: '%s'\n", workspaceModuleName)

	// Ensure DEPS/ directory exists, and warn if it seems to be mangled by the user.
	util.EnsureManagedDir(util.DepsDirName)

	workspaceModuleSymlink := ""
	if workspaceModuleFile.Layout != "cpp" {
		// Create the DEPS/ subdirectory and create a symlink to the top-level module.
		workspaceModuleSymlink = path.Join(workspaceRoot, util.DepsDirName, workspaceModuleName)
		if !util.DirExists(workspaceModuleSymlink) {
			log.Debug("Creating symlink for the workspace module: '%s/%s' -> '%s'.\n", util.DepsDirName, workspaceModuleName, workspaceRoot)
			util.MkdirAll(path.Dir(workspaceModuleSymlink))
			err := os.Symlink("..", workspaceModuleSymlink)
			if err != nil {
				log.Fatal("Failed to create symlink for workspace module: %s.\n", err)
			}
		}
	}

	errorFunc := func(format string, a ...interface{}) {
		log.Error(format, a...)
		log.Fatal("Use --ignore-errors to ignore this error.\n")
	}
	if ignoreErrors {
		errorFunc = log.Warning
	}

	// Modules that have been processed.
	done := map[string]bool{}

	// Modules that have been fetched.
	fetched := map[string]bool{}

	// Modules that still need to be processed.
	queue := []string{workspaceRoot}

	// Pinned dependency URLs / hashes.
	pinnedUrls := map[string]string{}
	pinnedHashes := map[string]string{}

	for len(queue) > 0 {
		modulePath := queue[0]
		queue = queue[1:]
		if done[modulePath] {
			continue
		}
		done[modulePath] = true

		moduleName := path.Base(modulePath)
		log.IndentationLevel = 0
		log.Log("Processing %s\n", moduleName)
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
			depModulePath := path.Join(workspaceRoot, util.DepsDirName, name)
			queue = append(queue, depModulePath)

			// Check that the dependency URL matches the pinned URL for that module.
			if _, isUrlPinned := pinnedUrls[name]; !isUrlPinned {
				pinnedUrls[name] = dep.URL
				log.Debug("Pinning URL to '%s'.\n", dep.URL)
			}
			if dep.URL != pinnedUrls[name] {
				errorFunc("Dependency requires URL '%s', but URL has been pinned to '%s'.\n", dep.URL, pinnedUrls[name])
			}

			// Check that the on-disk module has the same URL.
			depModule := module.OpenOrCreateModule(depModulePath, dep.URL, dep.Type, dep.Hash)
			if depModule.URL() != dep.URL {
				errorFunc("Dependency requires URL '%s', but the on-disk module has URL '%s'.\n", dep.URL, depModule.URL())
			}

			// Make sure we have the latest changes and the working tree is clean.
			if _, hasBeenFetched := fetched[depModulePath]; !hasBeenFetched {
				depModule.Fetch()
				fetched[depModulePath] = true
			}
			if depModule.IsDirty() {
				errorFunc("The exiting module has local changes.\n")
			}

			// Determine the commit hash for this dependency.

			// In --strict mode all hashes must be set in the MODULE file.
			if strict && dep.Hash == "" {
				errorFunc("Hash must not be empty in --strict mode.\n")
			}

			// Resolve the version string to a hash if we are currently processsing the
			// workspace module (only one module is "done") and the hash is not set yet or
			// --update is used to force re-resolution of the version string to a hash.
			if (update || dep.Hash == "") && len(done) == 1 {
				dep.Hash = depModule.RevParse(dep.Version)
				log.Debug("Resolved dependency version '%s' to hash '%s'.\n", dep.Version, dep.Hash[:7])
			}

			log.Log("Using hash '%s' for version '%s'.\n", dep.Hash[:7], dep.Version)

			// Check that the dependency hash is part of the tree that is referenced by the version string.
			if !depModule.IsAncestor(dep.Hash, dep.Version) {
				errorFunc(
					"The dependency hash ('%s') is not an ancestor of the commit ('%s') the version string ('%s') currently resolves to.\n",
					dep.Hash[:7], depModule.RevParse(dep.Version)[:7], dep.Version)
			}

			// Check the dependency hash against the fixed hash for that module.
			if _, isHashPinned := pinnedHashes[name]; !isHashPinned {
				pinnedHashes[name] = dep.Hash
			}
			pinnedHash := pinnedHashes[name]
			if dep.Hash != pinnedHash {
				errorFunc("Dependency requires hash '%s', but hash has been pinned to '%s'.\n", dep.Hash[:7], pinnedHash[:7])
			}

			// Check out the pinned hash.
			if depModule.Head() != pinnedHash {
				log.Log("Checking out '%s'.\n", pinnedHash[:7])
				depModule.Checkout(pinnedHash)
				module.SetupModule(depModulePath)
			}
			log.Log("\n")
		}
	}

	log.IndentationLevel = 0

	// Delete everything in the DEPS folder that does not belong there
	depsDir := path.Join(workspaceRoot, util.DepsDirName)
	content, err := ioutil.ReadDir(depsDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatal("%v", err)
	}

	if content != nil {
		for _, info := range content {
			fullPath := path.Join(depsDir, info.Name())
			if !done[fullPath] && fullPath != workspaceModuleSymlink && info.Name() != util.WarningFileName {
				log.Log("Deleting '%s'\n", fullPath)
				os.RemoveAll(fullPath)
			}
		}
	}

	if !strict {
		// Updated the MODULE file.
		for name, dep := range workspaceModuleFile.Dependencies {
			dep.Hash = pinnedHashes[name]
			workspaceModuleFile.Dependencies[name] = dep
		}
		module.WriteModuleFile(workspaceRoot, workspaceModuleFile)
	}

	log.Success("Done.\n")
}

func dependencyNames(file module.ModuleFile) []string {
	names := []string{}
	for name, dep := range file.Dependencies {
		modType := module.DetermineModuleType(dep.URL, dep.Type)

		if modType == module.GitModuleType {
			// Ensure that the dependency name matches the name of the git repo, since otherwise `module.Name()`
			// is broken.
			expectedName := strings.TrimSuffix(path.Base(dep.URL), ".git")
			if expectedName != name {
				log.Fatal("Dependency name does not match git repo name. Rename dependency %s to %s\n", name, expectedName)
			}
		}

		names = append(names, name)
	}
	return util.OrderedSlice(names)
}
