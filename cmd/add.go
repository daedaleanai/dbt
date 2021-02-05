package cmd

import (
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <repository> <version>",
	Args:  cobra.ExactArgs(2),
	Short: "Adds a dependency to the MODULE file of the current module",
	Long: `Adds a dependency to the MODULE file of the current module.
If the MODULE file already has an entry for the dependency, the version of the existing entry is updated.`,
	Run: runAdd,
}

func init() {
	depCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	log.Log("Current module is '%s'.\n", moduleRoot)

	deps := module.ReadModuleFile(moduleRoot)
	newDep := module.Dependency{
		URL:     args[0],
		Version: args[1],
	}

	for idx, dep := range deps {
		if dep.ModuleName() != newDep.ModuleName() {
			continue
		}
		if dep.URL != newDep.URL {
			log.Fatal("Module already has a dependency on a different module named '%s'.\n", newDep.ModuleName())
		}
		if dep.Version == newDep.Version {
			log.Success("Module already depends on module '%s', version '%s'.\n", newDep.ModuleName(), newDep.Version)
			return
		}
		deps[idx].Version = newDep.Version
		module.WriteModuleFile(moduleRoot, deps)
		log.Success("Updated version of dependency on module '%s' to '%s'.\n", newDep.ModuleName(), newDep.Version)
		return
	}

	module.WriteModuleFile(moduleRoot, append(deps, newDep))
	log.Success("Added dependency on module '%s', version '%s'.\n", newDep.ModuleName(), newDep.Version)
}
