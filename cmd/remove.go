package cmd

import (
	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"
)

var removeCmd = &cobra.Command{
	Use:   "remove MODULE",
	Args:  cobra.ExactArgs(1),
	Short: "Removes a dependency from the MODULE file of the current module",
	Long: `Removes a dependency from the MODULE file of the current module.
If the MODULE file does not have an entry for the dependency a working is printed.`,
	Run:               runRemove,
	ValidArgsFunction: completeRemoveArgs,
}

func init() {
	depCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	log.Log("Current module is '%s'.\n", moduleRoot)

	moduleFile := module.ReadModuleFile(moduleRoot)
	oldDepName := args[0]

	for idx, dep := range moduleFile.Dependencies {
		if dep.Name != oldDepName {
			continue
		}
		moduleFile.Dependencies = append(moduleFile.Dependencies[:idx], moduleFile.Dependencies[idx+1:]...)
		module.WriteModuleFile(moduleRoot, moduleFile)
		log.Success("Removed dependency on module '%s' from the current module.\n", oldDepName)
		return
	}

	log.Warning("The current module does not depend on module '%s'.\n", oldDepName)
}

func completeRemoveArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	moduleRoot := util.GetModuleRoot()
	moduleFile := module.ReadModuleFile(moduleRoot)

	suggestions := []string{}
	for _, dep := range moduleFile.Dependencies {
		suggestions = append(suggestions, dep.Name)
	}

	return suggestions, cobra.ShellCompDirectiveNoFileComp
}
