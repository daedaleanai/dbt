package cmd

import (
	"os"
	"path"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"

	"github.com/spf13/cobra"
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
	buildDir := path.Join(workspaceRoot, buildDirName)
	log.Debug("Removing '%s' diectory '%s'.\n", buildDirName, buildDir)
	os.RemoveAll(buildDir)
}
