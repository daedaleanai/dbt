package cmd

import (
	"dwm/log"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dwm",
	Short: "The Daedalean Workspace Manager (dwm)",
	Long: `The Daedalean Workspace Manager (dwm) helps setting up workspaces consisting
of multiple modules (git repositories) and managing dependencies between modules.
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.PersistentFlags().BoolVarP(&log.Verbose, "verbose", "v", false, "Print debug output")
	if rootCmd.Execute() != nil {
		os.Exit(1)
	}
}
