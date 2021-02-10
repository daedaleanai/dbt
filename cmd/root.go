package cmd

import (
	"os"

	"github.com/daedaleanai/dbt/log"

	"github.com/daedaleanai/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dbt",
	Short: "The Daedalean Build Tool (dbt)",
	Long: `The Daedalean Build Tool (dbt) helps setting up workspaces consisting
of multiple modules (git repositories), managing dependencies between modules, and
building build targets defined in those modules.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&log.Verbose, "verbose", "v", false, "Print debug output")
	if rootCmd.Execute() != nil {
		os.Exit(1)
	}
}
