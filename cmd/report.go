package cmd

import (
	"github.com/daedaleanai/cobra"
)

var reportCmd = &cobra.Command{
	Use:   "report [patterns] [build flags] [: test args]",
	Short: "Build selected reports.",
	Long:  `Build selected reports. If non-report targets specified report will be generated for selected targets.`,
	Run:   runReport,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeBuildArgs(toComplete, modeReport), cobra.ShellCompDirectiveNoFileComp
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(reportCmd)
	reportCmd.Flags().IntVarP(&numThreads, "threads", "j", -1, "Run N jobs in parallel. Defaults to as many threads as cores available.")
	reportCmd.Flags().IntVarP(&keepGoing, "keep", "k", 1, "Keep going until N jobs fail (0 means infinity)")
	reportCmd.Flags().SetInterspersed(false)
}

func runReport(cmd *cobra.Command, args []string) {
	testArgs := []string{}
	buildArgs := args
	for idx, arg := range args {
		if arg == ":" {
			testArgs = args[idx+1:]
			buildArgs = args[:idx]
			break
		}
	}
	runBuild(buildArgs, modeReport, testArgs)
}
