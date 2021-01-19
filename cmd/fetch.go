package cmd

import (
	"dbt/log"
	"dbt/module"
	"dbt/util"
	"io/ioutil"
	"path"

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

	depsDir := path.Join(workspaceRoot, util.DepsDirName)
	if !util.DirExists(depsDir) {
		log.Warning("There is no %s/ directory in the workspace. Maybe run 'dwm sync' first.\n", util.DepsDirName)
		return
	}
	files, err := ioutil.ReadDir(depsDir)
	if err != nil {
		log.Fatal("Failed to read content of %s/ directory: %s.\n", util.DepsDirName, err)
	}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		log.IndentationLevel = 0
		log.Log("\nProcessing module '%s'.\n", file.Name())
		log.IndentationLevel = 1
		mod := module.OpenModule(path.Join(depsDir, file.Name()))
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
