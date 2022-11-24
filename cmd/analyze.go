package cmd

import (
	"github.com/daedaleanai/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [patterns] [build flags] [: test args]",
	Short: "Builds the targets and generates static analysis reports.",
	Long:  `Builds the targets and generates static analysis reports.`,
	Run:   runAnalyze,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeBuildArgs(toComplete, modeAnalyze), cobra.ShellCompDirectiveNoFileComp
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
	analyzeCmd.Flags().SetInterspersed(false)
}

func runAnalyze(cmd *cobra.Command, args []string) {
	testArgs := []string{}
	buildArgs := args
	for idx, arg := range args {
		if arg == ":" {
			testArgs = args[idx+1:]
			buildArgs = args[:idx]
			break
		}
	}
	runBuild(buildArgs, modeAnalyze, testArgs)
}
