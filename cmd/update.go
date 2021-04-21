package cmd

import (
	"path"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update MODULES",
	Short: "Updates dependency version hashes the MODULE file of the current module",
	Long: `Updates dependency version hashes in MODULE file of the current module.
By default all modules in each MODULE file will be updated.
If a list of modules is supplied to the command only the hashes of the listed modules will be updated.
If the --all flag is provided, the MODULE files of all modules are updated.`,
	Run: runUpdate,
}

var allModules bool

func init() {
	updateCmd.Flags().BoolVar(&allModules, "all", false, "Update all modules")
	depCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	if allModules {
		workspaceRoot := util.GetWorkspaceRoot()
		log.Log("Current workspace is '%s'.\n", workspaceRoot)

		for modName, modPath := range module.GetAllModulePaths(workspaceRoot) {
			log.IndentationLevel = 0
			log.Log("\nProcessing module '%s'.\n", modName)
			log.IndentationLevel = 1
			updateModule(modPath, args)
		}
	} else {
		moduleRoot := util.GetModuleRoot()
		log.Log("Current module is '%s'.\n\n", moduleRoot)
		updateModule(moduleRoot, args)
	}

	log.IndentationLevel = 0
	log.Success("Done.\n")
}

func updateModule(moduleRoot string, modulesToUpdate []string) {
	// Whether to update the depndency to a given module.
	updateDependency := func(dep string) bool {
		if len(modulesToUpdate) == 0 {
			return true
		}
		for _, arg := range modulesToUpdate {
			if dep == arg {
				return true
			}
		}
		return false
	}

	depsDir := path.Join(util.GetWorkspaceRoot(), util.DepsDirName)
	changedModuleFile := false
	moduleFile := module.ReadModuleFile(moduleRoot)

	basicIndentationLevel := log.IndentationLevel
	for idx, dep := range moduleFile.Dependencies {
		log.IndentationLevel = basicIndentationLevel
		if !updateDependency(dep.Name) {
			continue
		}
		log.Log("%d) Updating dependency on module '%s', version '%s'.\n", idx+1, dep.Name, dep.Version.Rev)
		log.IndentationLevel = basicIndentationLevel + 1

		depModulePath := path.Join(depsDir, dep.Name)
		if !util.DirExists(depModulePath) {
			log.Warning("Dependency module does not exist. Run 'dbt sync' to create it. Not updating dependency version.\n")
			continue
		}

		depMod := module.OpenModule(depModulePath)
		if depMod.URL() != dep.URL {
			log.Warning("URL of existing module does not match dependency URL '%s'. Not updating dependency version.\n")
			continue
		}

		if depMod.IsDirty() {
			log.Warning("Dependency module has uncommited changes. Not updating version.\n")
			continue
		}

		newHash := depMod.RevParse(dep.Version.Rev)
		if dep.Version.Hash == newHash {
			log.Success("Version hash is already up-to-date.\n")
		} else {
			moduleFile.Dependencies[idx].Version.Hash = newHash
			changedModuleFile = true
			log.Success("Changed dependency version hash from '%s' to '%s'.\n", dep.Version.Hash, newHash)
		}

		log.Log("\n")
	}

	if changedModuleFile {
		module.WriteModuleFile(moduleRoot, moduleFile)
	}
}
