package cmd

import (
	"dwm/log"
	"dwm/module"
	"dwm/util"
	"os"

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
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) {
	moduleRoot, err := util.GetModuleRoot()
	if err != nil {
		log.Error("Could not identify module root directory. Make sure you run this command inside a module: %s.\n", err)
	}
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
			log.Error("Module already has a dependency on a different module named '%s'.\n", newDep.ModuleName())
		}
		if dep.Version == newDep.Version {
			log.Success("Module already depends on module '%s', version '%s'.\n", newDep.ModuleName(), newDep.Version)
			os.Exit(1)
		}
		deps[idx].Version = newDep.Version
		module.WriteModuleFile(moduleRoot, deps)
		log.Success("Updated version of dependency on module '%s' to '%s'.\n", newDep.ModuleName(), newDep.Version)
		os.Exit(0)
	}

	module.WriteModuleFile(moduleRoot, append(deps, newDep))
	log.Success("Added dependency on module '%s', version '%s'.\n", newDep.ModuleName(), newDep.Version)
}
