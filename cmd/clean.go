package cmd

import (
	"os"
	"path"

	"github.com/daedaleanai/dbt/v2/log"
	"github.com/daedaleanai/dbt/v2/util"

	"github.com/daedaleanai/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Args:  cobra.NoArgs,
	Short: "Removes all intermediate build results",
	Long:  `Removes all intermediate build results.`,
	Run:   runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}

func runClean(cmd *cobra.Command, args []string) {
	workspaceRoot := util.GetModuleRoot()
	log.Debug("Workspace: %s.\n", workspaceRoot)
	buildDir := path.Join(workspaceRoot, util.BuildDirName)
	log.Debug("Removing %s diectory '%s'.\n", util.BuildDirName, buildDir)
	os.RemoveAll(buildDir)
}
