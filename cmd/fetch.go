package cmd

import (
	"dbt/log"
	"dbt/module"
	"dbt/util"

	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Args:  cobra.NoArgs,
	Short: "Fetches remote changes for all currently checked out git dependencies",
	Long:  `Fetches remote changes for all currently checked out git dependencies.`,
	Run:   runFetch,
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}

func runFetch(cmd *cobra.Command, args []string) {
	workspaceRoot := util.GetWorkspaceRoot()
	log.Log("Current workspace is '%s'.\n", workspaceRoot)

	for modName, modPath := range module.GetAllModulePaths(workspaceRoot) {
		log.IndentationLevel = 0
		log.Log("\nProcessing module '%s'.\n", modName)
		log.IndentationLevel = 1
		mod := module.OpenModule(modPath)
		fetchedUpdates := mod.Fetch()
		if fetchedUpdates {
			log.Success("Fetched updates.\n")
		} else {
			log.Success("Already up-to-date.\n")
		}
	}

	log.IndentationLevel = 0
	log.Log("\n")
	log.Success("Done.\n")
}
