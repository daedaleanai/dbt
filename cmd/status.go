package cmd

import (
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
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
			log.Log("Current version hash: '%s'.\n", mod.Head())
		}
		deps := module.ReadModuleFile(mod.Path()).Dependencies
		log.Log("Module has %d dependencies.\n", len(deps))

		for idx, dep := range deps {
			log.IndentationLevel = 1
			log.Log("%d) Dependency on module '%s' (%s), version '%s' (%s).\n", idx+1, dep.Name, dep.URL, dep.Version.Rev, dep.Version.Hash)
			log.IndentationLevel = 2

			depMod, exists := modules[dep.Name]
			if !exists {
				log.Error("Dependency module does not exist. Try running 'dbt sync'.\n")
				continue
			}

			if depMod.URL() != dep.URL {
				log.Error("Dependency module URL does not match URL required by the dependency.\n")
				continue
			}

			if depMod.IsDirty() {
				log.Error("Dependency module has uncommited changes.\n")
				continue
			}

			if depMod.Head() != dep.Version.Hash {
				log.Error("Dependency module version does not match the required version.\n")
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
