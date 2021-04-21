package cmd

import (
	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/dbt/module"
)

var addTarCmd = &cobra.Command{
	Use:   "tar [NAME] URL",
	Args:  cobra.RangeArgs(1, 2),
	Short: "Adds a .tar.gz archive dependency to the MODULE file of the current module",
	Long:  `Adds a .tar.gz archive dependency to the MODULE file of the current module.`,
	Run:   runAddTar}

func init() {
	addCmd.AddCommand(addTarCmd)
}

func runAddTar(cmd *cobra.Command, args []string) {
	var dep module.Dependency
	if len(args) == 1 {
		dep = module.Dependency{
			Name:    parseNameFromUrl(args[0]),
			URL:     args[0],
			Version: module.Version{Rev: module.TarDefaultVersion, Hash: ""},
		}
	} else {
		dep = module.Dependency{
			Name:    args[0],
			URL:     args[1],
			Version: module.Version{Rev: module.TarDefaultVersion, Hash: ""},
		}
	}

	addDependency(dep)
}
