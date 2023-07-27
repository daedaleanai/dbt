package cmd

import (
	"path"
	"regexp"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"
)

var (
	nameRegexp    = regexp.MustCompile(`^[a-z0-9_\-.]+$`)
	urlRegexp     = regexp.MustCompile(`/([A-Za-z0-9_\-.]+)$`)
	versionRegexp = regexp.MustCompile(`^[A-Za-z0-9_\-./]+$`)
	hashRegexp    = regexp.MustCompile(`^[A-Za-z0-9]+$`)
)

const masterVersion = "origin/master"

var (
	depCmd = &cobra.Command{
		Use:   "dep",
		Short: "Manages module dependencies",
	}

	addCmd = &cobra.Command{
		Use:               "add [NAME] --url=URL [--version=VERSION]",
		Args:              cobra.RangeArgs(0, 1),
		Short:             "Adds a dependency to the MODULE file of the current module",
		Long:              `Adds a dependency to the MODULE file of the current module.`,
		Run:               runAdd,
		ValidArgsFunction: completeDepArgs,
	}

	removeCmd = &cobra.Command{
		Use:               "remove MODULE",
		Args:              cobra.ExactArgs(1),
		Short:             "Removes a dependency from the MODULE file of the current module",
		Long:              `Removes a dependency from the MODULE file of the current module.`,
		Run:               runRemove,
		ValidArgsFunction: completeDepArgs,
	}

	overrideCmd = &cobra.Command{
		Use:               "override MODULE",
		Args:              cobra.ExactArgs(1),
		Short:             "Overrides a module",
		Long:              `Overrides a module.`,
		Run:               runOverride,
		ValidArgsFunction: completeDepArgs,
	}
)

var url, version, hash, depType string

func init() {
	rootCmd.AddCommand(depCmd)

	depCmd.AddCommand(addCmd)
	addCmd.Flags().StringVar(&url, "url", "", "Dependency URL")
	addCmd.Flags().StringVar(&version, "version", masterVersion, "Dependency version")
	addCmd.Flags().StringVar(&hash, "hash", "", "Dependency hash")
	addCmd.Flags().StringVar(&depType, "type", "", "Dependency type")

	depCmd.AddCommand(removeCmd)

	depCmd.AddCommand(overrideCmd)
	overrideCmd.Flags().StringVar(&url, "url", "", "Dependency URL")
	overrideCmd.Flags().StringVar(&version, "version", masterVersion, "Dependency version")
	overrideCmd.Flags().StringVar(&hash, "hash", "", "Dependency hash")
	overrideCmd.Flags().StringVar(&depType, "type", "", "Dependency type")
}

func runAdd(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	moduleName := path.Base(moduleRoot)
	log.Debug("Module: %s.\n", moduleRoot)

	moduleFile := module.ReadModuleFile(moduleRoot)

	var name string
	if len(args) == 0 {
		checkUrl(url)
		name = urlRegexp.FindStringSubmatch(url)[1]
	} else {
		name = args[0]
	}
	checkName(name)

	dep, exists := moduleFile.Dependencies[name]
	if url != "" {
		dep.URL = url
	}
	if version != "" {
		dep.Version = version
	}

	checkUrl(dep.URL)
	checkVersion(dep.Version)

	moduleFile.Dependencies[name] = dep
	module.WriteModuleFile(moduleRoot, moduleFile)

	if exists {
		log.Success("Updated dependency '%s' to module '%s'.\n", name, moduleName)
		log.Debug("Updated dependency '%s' to module '%s': URL='%s', version='%s'.\n", name, moduleName, url, version)
	} else {
		log.Success("Added dependency '%s' to module '%s'.\n", name, moduleName)
		log.Debug("Added dependency '%s' to module '%s': URL='%s', version='%s'.\n", name, moduleName, url, version)
	}
}

func runOverride(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	moduleName := path.Base(moduleRoot)
	log.Debug("Module: %s.\n", moduleRoot)

	moduleFile := module.ReadModuleFile(moduleRoot)

	var name string
	if len(args) == 0 {
		checkUrl(url)
		name = urlRegexp.FindStringSubmatch(url)[1]
	} else {
		name = args[0]
	}
	checkName(name)

	dep, exists := moduleFile.Overrides[name]
	if url == "" {
		log.Fatal("Unspecified URL. use `--url $URL` to specify an URL of the dependency to override")
	}
	if hash == "" {
		log.Fatal("Unspecified hash. use `--hash $HASH` to specify a hash of the dependency to override")
	}
	dep.URL = url
	dep.Version = version
	dep.Hash = hash
	dep.Type = depType

	checkUrl(dep.URL)
	checkVersion(dep.Version)
	checkHash(dep.Hash)
	checkDepType(dep.URL, dep.Type)

	if moduleFile.Overrides == nil {
		moduleFile.Overrides = map[string]module.Dependency{}
	}
	moduleFile.Overrides[name] = dep
	module.WriteModuleFile(moduleRoot, moduleFile)

	if exists {
		log.Success("Updated override '%s' to module '%s'.\n", name, moduleName)
		log.Debug("Updated override '%s' to module '%s': URL='%s', version='%s', hash='%s', type='%s'.\n", name, moduleName, dep.URL, dep.Version, dep.Hash, dep.Type)
	} else {
		log.Success("Added override '%s' to module '%s'.\n", name, moduleName)
		log.Debug("Added override '%s' to module '%s': URL='%s', version='%s', hash='%s', type='%s'.\n", name, moduleName, dep.URL, dep.Version, dep.Hash, dep.Type)
	}
}

func runRemove(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	moduleName := path.Base(moduleRoot)
	log.Debug("Module: '%s'.\n", moduleRoot)

	moduleFile := module.ReadModuleFile(moduleRoot)
	name := args[0]
	checkName(name)

	if _, exists := moduleFile.Dependencies[name]; !exists {
		log.Warning("Module '%s' has no dependency on module '%s'.\n", moduleName, name)
		return
	}

	delete(moduleFile.Dependencies, name)
	module.WriteModuleFile(moduleRoot, moduleFile)
	log.Success("Removed dependency '%s' from module '%s'.\n", name, moduleName)
}

func checkName(name string) {
	if !nameRegexp.MatchString(name) {
		log.Fatal("Module name '%s' does not match the expected format.\n", url)
	}
}

func checkUrl(url string) {
	if !urlRegexp.MatchString(url) {
		log.Fatal("URL '%s' does not match the expected format.\n", url)
	}
}

func checkVersion(version string) {
	if !versionRegexp.MatchString(version) {
		log.Fatal("Version '%s' does not match the expected format.\n", version)
	}
}

func checkHash(hash string) {
	if !hashRegexp.MatchString(hash) {
		log.Fatal("Hash '%s' does not match the expected format.\n", hash)
	}
}

func checkDepType(url, depType string) {
	_, error := module.DetermineModuleType(url, depType)
	if error != nil {
		log.Fatal("%s", error.Error())
	}
}

func completeDepArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	completions := []string{}
	if len(args) == 0 {
		moduleRoot := util.GetModuleRoot()
		moduleFile := module.ReadModuleFile(moduleRoot)
		for name := range moduleFile.Dependencies {
			completions = append(completions, name)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
