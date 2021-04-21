package cmd

import (
	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/dbt/module"
)

var addGitCmd = &cobra.Command{
	Use:   "git [NAME] URL VERSION",
	Args:  cobra.RangeArgs(2, 3),
	Short: "Adds a git repository dependency to the MODULE file of the current module",
	Long:  `Adds a git repository dependency to the MODULE file of the current module.`,
	Run:   runAddGit}

func init() {
	addCmd.AddCommand(addGitCmd)
}

func runAddGit(cmd *cobra.Command, args []string) {
	var dep module.Dependency
	if len(args) == 2 {
		dep = module.Dependency{
			Name:    parseNameFromUrl(args[0]),
			URL:     args[0],
			Version: module.Version{Rev: args[1], Hash: ""},
		}
	} else {
		dep = module.Dependency{
			Name:    args[0],
			URL:     args[1],
			Version: module.Version{Rev: args[2], Hash: ""},
		}
	}

	addDependency(dep)
}
