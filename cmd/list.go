package cmd

import (
	"github.com/daedaleanai/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [build flags]",
	Short: "Lists all targets.",
	Long:  `Lists all targets.`,
	Run:   runList,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeBuildArgs(toComplete, modeList), cobra.ShellCompDirectiveNoFileComp
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	runBuild(args, modeList, nil)
}
