package cmd

import (
	"github.com/daedaleanai/cobra"
)

var flagsCmd = &cobra.Command{
	Use:   "flags [build flags]",
	Short: "Lists all flags",
	Long:  `Lists all flags and optionally updates any/all of them.`,
	Run:   runFlags,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeBuildArgs(toComplete, modeList), cobra.ShellCompDirectiveNoFileComp
	},
	DisableFlagsInUseLine: false,
}

func init() {
	rootCmd.AddCommand(flagsCmd)
}

func runFlags(cmd *cobra.Command, args []string) {
	runBuild(args, modeFlags, nil)
}
