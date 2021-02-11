package cmd

import (
	"github.com/daedaleanai/cobra"
)

var depCmd = &cobra.Command{
	Use:   "dep",
	Short: "Manages module dependencies",
}

func init() {
	rootCmd.AddCommand(depCmd)
}
