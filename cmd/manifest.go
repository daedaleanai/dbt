package cmd

import (
	"path"
	"sort"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

var manifestCmd = &cobra.Command{
	Use:   "manifest",
	Args:  cobra.NoArgs,
	Short: "Generates a manifest file containing all dependencies and their versions",
	Long:  `Generates a manifest file containing all dependencies and their versions, as currently synced.`,
	Run:   runManifest,
}

var manifestAllowUncommittedChanges bool
var manifestOutput string

func init() {
	manifestCmd.Flags().BoolVar(&manifestAllowUncommittedChanges, "allow-uncommitted-changes", false, "Continues even if there are local uncommitted changes.")
	manifestCmd.Flags().StringVar(&manifestOutput, "o", "manifest.yaml", "File where the manifest will be stored")
	rootCmd.AddCommand(manifestCmd)
}

func runManifest(cmd *cobra.Command, args []string) {
	workspaceRoot := util.GetWorkspaceRoot()
	log.Debug("Workspace: %s.\n", workspaceRoot)

	workspaceModule := module.OpenModule(workspaceRoot)
	workspaceModuleName := workspaceModule.Name()
	log.Debug("Workspace module name: '%s\n", workspaceModuleName)

	dependencyNames := func(file *module.ModuleFile) []string {
		names := []string{}
		for name := range file.Dependencies {
			names = append(names, name)
		}
		sort.Strings(names)
		return names
	}

	type ModuleManifest struct {
		Name string
		Url  string
		Hash string
	}

	// Modules that have been processed.
	processed := map[string]ModuleManifest{}

	// Modules that still need to be processed.
	queue := []string{workspaceRoot}

	for len(queue) > 0 {
		modulePath := queue[0]
		queue = queue[1:]
		if _, ok := processed[modulePath]; ok {
			continue
		}

		moduleName := path.Base(modulePath)
		log.IndentationLevel = 0
		log.Debug("Processing %s\n", moduleName)
		log.IndentationLevel = 1

		currentModule := module.OpenModule(modulePath)
		currentModuleFile := module.ReadModuleFile(modulePath)

		if currentModule.IsDirty() && !manifestAllowUncommittedChanges {
			log.Fatal("Module '%s' contains uncommitted changes\n", currentModule.Name())
		}

		// Store current module
		processed[modulePath] = ModuleManifest{
			Name: currentModule.Name(),
			Url:  currentModule.URL(),
			Hash: currentModule.Head(),
		}

		if len(currentModuleFile.Dependencies) == 0 {
			log.IndentationLevel = 1
			log.Debug("Has no dependencies\n\n")
			continue
		}

		for _, name := range dependencyNames(&currentModuleFile) {
			log.IndentationLevel = 1
			log.Debug("Depends on %s\n", name)
			log.IndentationLevel = 2

			depModulePath := path.Join(workspaceRoot, util.DepsDirName, name)
			queue = append(queue, depModulePath)
		}
	}

	modules := []ModuleManifest{}
	for _, mod := range processed {
		modules = append(modules, mod)
	}
	util.WriteYaml(manifestOutput, modules)

	log.IndentationLevel = 0
	log.Success("Done.\n")
}
