package cmd

import (
	"dbt/log"
	"dbt/module"
	"dbt/util"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Args:  cobra.NoArgs,
	Short: "Prints a status report of all checked-out modules and their dependencies",
	Long:  `Prints a status report of all checked-out modules and their dependencies.`,
	Run:   runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	workspaceRoot := util.GetWorkspaceRoot()
	log.Log("Workspace: '%s'\n", workspaceRoot)

	modules := map[string]module.Module{}
	for _, modPath := range module.GetAllModulePaths(workspaceRoot) {
		mod := module.OpenModule(modPath)
		modules[mod.Name()] = mod
	}

	for _, mod := range modules {
		log.IndentationLevel = 0
		log.Log("\nChecking module '%s':\n", mod.Name())
		log.IndentationLevel = 1

		if mod.IsDirty() {
			log.Error("Module has uncommited changes.\n")
		} else {
			log.Log("Current versions: '%s'.\n", strings.Join(mod.CheckedOutVersions(), "', '"))
		}
		deps := module.ReadModuleFile(mod.Path())
		log.Log("Module has %d dependencies.\n", len(deps))

		for idx, dep := range deps {
			log.IndentationLevel = 1
			log.Log("%d) Dependency on module '%s' (%s), version '%s':\n", idx+1, dep.ModuleName(), dep.URL, dep.Version)
			log.IndentationLevel = 2

			depMod, exists := modules[dep.ModuleName()]
			if !exists {
				log.Error("Dependency module does not exist. Try running 'dbt sync'.\n")
				continue
			}

			if !depMod.HasOrigin(dep.URL) {
				log.Error("Dependency module origin does not match URL required by the dependency.\n")
				continue
			}

			if depMod.IsDirty() {
				log.Error("Dependency module has uncommited changes.\n")
				continue
			}

			if !depMod.HasVersionCheckedOut(dep.Version) {
				versions := depMod.CheckedOutVersions()
				log.Error("Dependency module version does not match the required version. Current versions are: '%s'.\n", strings.Join(versions, "', '"))
				continue
			}

			log.Success("Dependency is fulfilled.\n")
		}
	}

	log.IndentationLevel = 0
	log.Log("\n")
	if log.ErrorOccured() {
		log.Fatal("Errors found while checking workspace status.\n")
	}
	log.Success("Done.\n")
}
