package cmd

import (
	"fmt"
	"os"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "dbt",
		Short: "The Daedalean Build Tool (dbt)",
		Long: `The Daedalean Build Tool (dbt) helps setting up workspaces consisting
of multiple modules (git repositories), managing dependencies between modules, and
building build targets defined in those modules.`,
		Version: fmt.Sprintf("v%d.%d.%d", util.DbtVersion[0], util.DbtVersion[1], util.DbtVersion[2]),
	}
	noWorkspaceChecks = false
)

func init() {
	cobra.OnInitialize(initWorkspace)

	rootCmd.PersistentFlags().BoolVarP(&log.Verbose, "verbose", "v", false, "print debug output")
	rootCmd.PersistentFlags().BoolVar(&log.NoColor, "no-color", false, "does not colorize the output")
	rootCmd.PersistentFlags().BoolVar(&noWorkspaceChecks, "no-workspace-checks", false,
		"DANGEROUS: skip checks that the special purpose directories (BUILD, DEPS) are not adjusted by the user")
}

func initWorkspace() {
	if noWorkspaceChecks {
		return
	}
	util.CheckManagedDirs()
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if rootCmd.Execute() != nil {
		os.Exit(1)
	}
}
