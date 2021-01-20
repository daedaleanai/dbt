package cmd

import (
	"dbt/log"
	"dbt/module"
	"dbt/util"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update [<modules>]",
	Short: "Updates dependency versions in MODULE files",
	Long: `Updates dependency versions in MODULE files. By default only the current module will be updated.
If the --global option is present, all MODULE files in the current workspace will be updated.
By default all modules in each MODULE file will be updated.
If a list of modules is supplied to the command only the versions of the listed modules will be updated.
The new version of a dependency is determined by taking the HEAD of the currently checked out module repository.
If the head has an annotated tag associated with it, the tag name is used.
Otherwise, the commit hash is used.`,
	Run: runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) {
	updateDependency := func(dep string) bool {
		if len(args) == 0 {
			return true
		}
		for _, arg := range args {
			if dep == arg {
				return true
			}
		}
		return false
	}

	moduleRoot := util.GetModuleRoot()
	log.Log("Current module is '%s'.\n", moduleRoot)

	depsDir := path.Join(util.GetWorkspaceRoot(), util.DepsDirName)
	deps := module.ReadModuleFile(moduleRoot)

	for idx, dep := range deps {
		log.IndentationLevel = 0
		if !updateDependency(dep.ModuleName()) {
			continue
		}

		log.Log("\nUpdating dependency on module '%s', version '%s':\n", dep.ModuleName(), dep.Version)
		log.IndentationLevel = 1

		depModulePath := path.Join(depsDir, dep.ModuleName())
		if !util.DirExists(depModulePath) {
			log.Warning("Dependency module does not exist. Run 'dwm sync' to create it. Not updating dependency version.\n")
			continue
		}

		depMod := module.OpenModule(depModulePath)
		if !depMod.HasOrigin(dep.URL) {
			log.Warning("Origin of existing module does not match dependency URL '%s'. Not updating dependency version.\n")
			continue
		}

		if depMod.HasVersionCheckedOut(dep.Version) {
			log.Success("Dependency version is already up-to-date.\n")
			continue
		}

		versions := depMod.CheckedOutVersions()
		if len(versions) == 0 {
			log.Warning("Dependency module has uncommited changes. Not updating version.\n")
			continue
		}
		newVersion := versions[len(versions)-1]
		if len(versions) > 1 {
			log.Debug("Dependency module has multiple versions: '%s'. Using '%s'.\n", strings.Join(versions, "', '"), newVersion)
		}
		log.Success("Changing dependency version from '%s' to '%s'.\n", dep.Version, newVersion)
		deps[idx].Version = newVersion
	}

	module.WriteModuleFile(moduleRoot, deps)

	log.IndentationLevel = 0
	log.Log("\n")
	log.Success("Done.\n")
}
