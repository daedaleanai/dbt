package cmd

import (
	"github.com/daedaleanai/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [patterns] [build flags] [: run args]",
	Short: "Builds and runs the targets",
	Long:  `Builds and runs the targets.`,
	Run:   runRun,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeBuildArgs(toComplete, modeRun), cobra.ShellCompDirectiveNoFileComp
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().IntVarP(&numThreads, "threads", "j", -1, "Run N jobs in parallel. Defaults to as many threads as cores available.")
	runCmd.Flags().IntVarP(&keepGoing, "keep", "k", 1, "Keep going until N jobs fail (0 means infinity)")
	runCmd.Flags().SetInterspersed(false)
}

func runRun(cmd *cobra.Command, args []string) {
	runArgs := []string{}
	buildArgs := args
	for idx, arg := range args {
		if arg == ":" {
			runArgs = args[idx+1:]
			buildArgs = args[:idx]
			break
		}
	}
	runBuild(buildArgs, modeRun, runArgs)
}
