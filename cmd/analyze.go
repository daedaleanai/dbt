package cmd

import (
	"github.com/daedaleanai/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze [patterns] [build flags] [: test args]",
	Short: "Builds, tests the targets and generate static analysis report.",
	Long:  `Builds, tests the targets and generate static analysis report.`,
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
	forceRegenerate = true

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
