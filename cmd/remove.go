package cmd

import (
	"dbt/log"
	"dbt/module"
	"dbt/util"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <repository|module>",
	Args:  cobra.ExactArgs(1),
	Short: "Removes a dependency from the MODULE file of the current module",
	Long: `Removes a dependency from the MODULE file of the current module.
If the MODULE file does not have an entry for the dependency a working is printed.`,
	Run: runRemove,
}

func init() {
	depCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	log.Log("Current module is '%s'.\n", moduleRoot)

	deps := module.ReadModuleFile(moduleRoot)
	oldDep := args[0]

	var found = false
	for idx, dep := range deps {
		if dep.ModuleName() == oldDep || dep.URL == oldDep {
			oldDep = dep.ModuleName()
			deps = append(deps[:idx], deps[idx+1:]...)
			found = true
		}
	}

	if found {
		module.WriteModuleFile(moduleRoot, deps)
		log.Success("Removed dependency on module '%s' from module.\n", oldDep)
	} else {
		log.Warning("The current module does not depend on module '%s'.\n", oldDep)
	}
}
