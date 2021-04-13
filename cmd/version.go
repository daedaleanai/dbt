package cmd

import (
	"fmt"

	"github.com/daedaleanai/cobra"
)

const dbtVersion = "v0.4.2"

var versionCmd = &cobra.Command{
	Use:   "version",
	Args:  cobra.NoArgs,
	Short: "Prints the version of this tool",
	Long:  `Prints the version of this tool.`,
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("dbt %s\n", dbtVersion)
}
