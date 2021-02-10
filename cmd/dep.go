package cmd

import (
	"github.com/daedaleanai/cobra"
)

var depCmd = &cobra.Command{
	Use: "dep",
}

func init() {
	rootCmd.AddCommand(depCmd)
}
