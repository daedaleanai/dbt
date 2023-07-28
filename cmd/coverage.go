package cmd

import (
	"github.com/daedaleanai/cobra"
)

var coverageCmd = &cobra.Command{
	Use:   "coverage [patterns] [build flags] [: test args]",
	Short: "Builds, tests the targets and generate coverage report.",
	Long:  `Builds, tests the targets and generate coverage report.`,
	Run:   runCoverage,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeBuildArgs(toComplete, modeCoverage), cobra.ShellCompDirectiveNoFileComp
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(coverageCmd)
	coverageCmd.Flags().IntVarP(&numThreads, "threads", "j", -1, "Run N jobs in parallel")
	coverageCmd.Flags().SetInterspersed(false)
}

func runCoverage(cmd *cobra.Command, args []string) {
	numThreads = 1
	testArgs := []string{}
	buildArgs := args
	for idx, arg := range args {
		if arg == ":" {
			testArgs = args[idx+1:]
			buildArgs = args[:idx]
			break
		}
	}
	runBuild(buildArgs, modeCoverage, testArgs)
}
