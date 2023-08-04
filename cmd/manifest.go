package cmd

import (
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/manifest"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Args:  cobra.NoArgs,
	Short: "Generates or diffs dbt manifests",
	Long:  `Generates or diffs dbt manifests.`,
}

var manifestAllowUncommittedChanges bool
var manifestOutput string

func init() {
	diffCommand := &cobra.Command{
		Use:   "diff [newManifest] oldManifest",
		Args:  cobra.ArbitraryArgs,
		Short: "Diffs two manifests and lists their differences per module",
		Long:  `Diffs two manifests and lists their differences per module. If [newManifest] is omitted, then the command will show the differences respect to the current working revision.`,
		Run:   runManifestDiff,
	}
	manifestCmd.AddCommand(diffCommand)

	generateCommand := &cobra.Command{
		Use:   "generate",
		Args:  cobra.NoArgs,
		Short: "Creates a manifest file storing the currently synced state of the repository",
		Long:  `Creates a manifest file storing the currently synced state of the repository.`,
		Run:   runManifestGenerate,
	}
	generateCommand.Flags().BoolVar(&manifestAllowUncommittedChanges, "allow-uncommitted-changes", false, "Continues even if there are local uncommitted changes.")
	generateCommand.Flags().StringVarP(&manifestOutput, "output", "o", "manifest.yaml", "File where the manifest will be stored")

	manifestCmd.AddCommand(generateCommand)
	rootCmd.AddCommand(manifestCmd)
}

func runManifestDiff(cmd *cobra.Command, args []string) {
	var manifestNewPath string = "<HEAD>"
	var manifestOldPath string

	var manifestNew manifest.Manifest
	var manifestOld manifest.Manifest

	if len(args) == 1 {
		// One file provided, assume that we want to diff this manifest against the currently synced state
		workspaceRoot := util.GetWorkspaceRoot()
		var err error

		manifestNew, err = manifest.Generate(module.GetAllModules(workspaceRoot), true)
		if err != nil {
			// This is never expected to fail when allowUncommittedChanges is true, but in case it does...
			log.Fatal("manifest.Generate failed unexpectedly: %s\n", err.Error())
		}

		manifestOldPath = args[0]
		util.ReadYaml(manifestOldPath, &manifestOld)
	} else if len(args) == 2 {
		manifestNewPath = args[0]
		manifestOldPath = args[1]
		util.ReadYaml(manifestNewPath, &manifestNew)
		util.ReadYaml(manifestOldPath, &manifestOld)
	} else {
		log.Fatal("\"dbt manifest diff\" takes either one argument or two arguments.\n")
	}

	diff, err := manifest.Diff(manifestNew, manifestOld)
	if err != nil {
		log.Fatal("Error parsing diff between manifests: %s\n", err.Error())
	}

	log.IndentationLevel = 0

	if !diff.Differ {
		log.Log("Manifests are identical.\n")
		return
	}

	// Print the diff information
	if diff.DbtVersion != "" {
		log.Log("%s\n", diff.DbtVersion)
	}

	if len(diff.AddedModules) != 0 {
		log.Log("Added modules:\n")
		for _, addedMod := range diff.AddedModules {
			log.IndentationLevel = 1
			log.Log("%s:\n", addedMod.Name)
			log.IndentationLevel = 2
			log.Log("URL: %s\n", addedMod.Url)
			log.Log("Hash: %s\n", addedMod.Hash)
			log.Log("Type: %s\n", addedMod.Type)
			log.IndentationLevel = 0
		}
		log.Log("\n")
	}

	if len(diff.RemovedModules) != 0 {
		log.Log("Removed modules:\n")
		for _, removedMod := range diff.RemovedModules {
			log.IndentationLevel = 1
			log.Log("%s:\n", removedMod.Name)
			log.IndentationLevel = 2
			log.Log("URL: %s\n", removedMod.Url)
			log.Log("Hash: %s\n", removedMod.Hash)
			log.Log("Type: %s\n", removedMod.Type)
			log.IndentationLevel = 0
		}
		log.Log("\n")
	}

	if len(diff.ModifiedModules) != 0 {
		log.Log("Modified modules:\n")
		for _, modifiedMod := range diff.ModifiedModules {
			log.IndentationLevel = 1
			log.Log("%s:\n", modifiedMod.New.Name)
			log.IndentationLevel = 2
			if modifiedMod.New.Url != modifiedMod.Old.Url {
				log.Log("URL changed from %q to %q\n", modifiedMod.Old.Url, modifiedMod.New.Url)
			}
			if modifiedMod.New.Hash != modifiedMod.Old.Hash {
				log.Log("Hash changed from %q to %q\n", modifiedMod.Old.Hash, modifiedMod.New.Hash)
				log.IndentationLevel = 3

				if modifiedMod.FirstCommonAncestor != nil {
					log.Log("Common ancestor: %v\n", modifiedMod.FirstCommonAncestor.String())
				}

				const RED_DASH string = "\u001b[31;1m-\u001b[0m"
				const GREEN_PLUS string = "\u001b[32;1m+\u001b[0m"

				if len(modifiedMod.AddedCommits) != 0 {
					log.Log("Added commits: \n")
					log.IndentationLevel = 4
					for _, commit := range modifiedMod.AddedCommits {
						log.Log("%s %s\n", GREEN_PLUS, commit.String())
					}
					log.IndentationLevel = 3
				}

				if len(modifiedMod.DiscardedCommits) != 0 {
					log.Log("Discarded commits: \n")
					log.IndentationLevel = 4
					for _, commit := range modifiedMod.DiscardedCommits {
						log.Log("%s %s\n", RED_DASH, commit.String())
					}
					log.IndentationLevel = 3
				}

				log.IndentationLevel = 2
			}
			if modifiedMod.New.Type != modifiedMod.Old.Type {
				log.Log("Type changed from %q to %q\n", modifiedMod.Old.Type, modifiedMod.New.Type)
			}
			if modifiedMod.New.Dirty != modifiedMod.Old.Dirty {
				if modifiedMod.New.Dirty {
					log.Log("New module is dirty\n")
				} else {
					log.Log("Old module is dirty\n")
				}
			}

			log.IndentationLevel = 0
		}

		log.Log("\n")
	}
}

func runManifestGenerate(cmd *cobra.Command, args []string) {
	workspaceRoot := util.GetWorkspaceRoot()

	manifest, err := manifest.Generate(module.GetAllModules(workspaceRoot), manifestAllowUncommittedChanges)
	if err != nil {
		log.Fatal("%s\n", err)
	}

	util.WriteYaml(manifestOutput, manifest)
	log.Success("Done.\n")
}
