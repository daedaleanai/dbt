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
