package cmd

import (
	"os"
	"path"
	"strings"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/dbt/v2/log"
	"github.com/daedaleanai/dbt/v2/module"
	"github.com/daedaleanai/dbt/v2/util"
)

var cloneCmd = &cobra.Command{
	Use:   "clone URL [repo_path]",
	Args:  cobra.RangeArgs(1, 2),
	Short: "Clones a repository locally, recursively syncing and updating modules to satisfy all dependencies.",
	Long: `Clones a repository locally, recursively syncing and updating modules to satisfy the dependencies
declared in the MODULE files of each module, starting from the top-level MODULE file.`,
	Run: runClone,
}

var revision string
var mirror bool

func init() {
	cloneCmd.Flags().StringVar(&revision, "revision", "master", "The revision to clone.")
	rootCmd.AddCommand(cloneCmd)
}

func runClone(cmd *cobra.Command, args []string) {
	repoUrl := args[0]
	repoPath := ""
	if len(args) > 1 {
		if path.IsAbs(args[1]) {
			repoPath = args[1]
		} else {
			repoPath = path.Join(util.GetWorkingDir(), args[1])
		}
	} else {
		parts := strings.Split(repoUrl, "/")
		repoName := parts[len(parts)-1]
		repoName = strings.TrimSuffix(repoName, ".git")
		repoPath = path.Join(util.GetWorkingDir(), repoName)
	}

	if util.DirExists(repoPath) {
		log.Error("Directory '%s' already exists.\n", repoPath)
		return
	}

	log.Log("Cloning '%s' into '%s'.\n", repoUrl, repoPath)
	mod, err := module.CreateGitModule(repoPath, repoUrl)
	if err != nil {
		os.RemoveAll(repoPath)
		log.Fatal("Failed to create git module: %s.\n", err)
	}
	log.Log("Checking out revision '%s'\n", revision)
	mod.Checkout(revision)
	module.SetupModule(repoPath)

	// Move into the repo directory in order to set it up
	os.Chdir(repoPath)
	runSync(cmd, args)
}
