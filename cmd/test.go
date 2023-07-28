package cmd

import (
	"github.com/daedaleanai/cobra"
)

var testCmd = &cobra.Command{
	Use:   "test [patterns] [build flags] [: test args]",
	Short: "Builds and tests the targets",
	Long:  `Builds and tests the targets.`,
	Run:   runTest,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeBuildArgs(toComplete, modeTest), cobra.ShellCompDirectiveNoFileComp
	},
	DisableFlagsInUseLine: true,
}

func init() {
	rootCmd.AddCommand(testCmd)
	testCmd.Flags().IntVarP(&numThreads, "threads", "j", -1, "Run N jobs in parallel")
	testCmd.Flags().SetInterspersed(false)
}

func runTest(cmd *cobra.Command, args []string) {
	testArgs := []string{}
	buildArgs := args
	for idx, arg := range args {
		if arg == ":" {
			testArgs = args[idx+1:]
			buildArgs = args[:idx]
			break
		}
	}
	runBuild(buildArgs, modeTest, testArgs)
}
