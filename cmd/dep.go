package cmd

import (
	"regexp"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"
)

var (
	nameRegexp = regexp.MustCompile(`^[a-z0-9_\-.]+$`)
	urlRegexp  = regexp.MustCompile(`/([A-Za-z0-9_\-.]+)(\.git|\.tar\.gz)$`)
	revRegexp  = regexp.MustCompile(`^[A-Za-z0-9_\-.]+$`)
)

var (
	depCmd = &cobra.Command{
		Use:   "dep",
		Short: "Manages module dependencies",
	}

	addCmd = &cobra.Command{
		Use:   "add [NAME] --url=URL [--version=VERSION]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "Adds a dependency to the MODULE file of the current module",
		Long:  `Adds a dependency to the MODULE file of the current module.`,
		Run:   runAdd,
	}

	updateCmd = &cobra.Command{
		Use:               "update NAME [--url=URL] [--version=VERSION]",
		Args:              cobra.RangeArgs(1, 1),
		Short:             "Updates a dependency in the MODULE file of the current module",
		Long:              `Updates a dependency in the MODULE file of the current module.`,
		ValidArgsFunction: completeDepArgs,
		Run:               runUpdate,
	}

	removeCmd = &cobra.Command{
		Use:               "remove MODULE",
		Args:              cobra.ExactArgs(1),
		Short:             "Removes a dependency from the MODULE file of the current module",
		Long:              `Removes a dependency from the MODULE file of the current module.`,
		Run:               runRemove,
		ValidArgsFunction: completeDepArgs,
	}
)

var url, version string

func init() {
	rootCmd.AddCommand(depCmd)

	depCmd.AddCommand(addCmd)
	addCmd.Flags().StringVar(&url, "url", "", "Dependency URL")
	addCmd.Flags().StringVar(&version, "version", "master", "Dependency version")

	depCmd.AddCommand(updateCmd)
	updateCmd.Flags().StringVar(&url, "url", "", "Dependency URL")
	updateCmd.Flags().StringVar(&version, "version", "", "Dependency version")

	depCmd.AddCommand(removeCmd)
}

func runAdd(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	log.Debug("Current module is '%s'.\n", moduleRoot)
	moduleFile := module.ReadModuleFile(moduleRoot)

	checkUrl(url)
	checkRev(version)

	var name string
	if len(args) == 0 {
		name = urlRegexp.FindStringSubmatch(url)[1]
	} else {
		name = args[0]
	}
	checkName(name)

	for _, dep := range moduleFile.Dependencies {
		if dep.Name == name {
			log.Fatal("Dependency '%s' already exists. Try 'dbt dep update'.\n", name)
		}
	}

	moduleFile.Dependencies = append(moduleFile.Dependencies, module.Dependency{
		Name:    name,
		URL:     url,
		Version: module.Version{Rev: version, Hash: ""},
	})
	module.WriteModuleFile(moduleRoot, moduleFile)
	log.Success("Added dependency '%s'.\n", name)
	log.Debug("Added dependency '%s': URL='%s', version='%s'.\n", name, url, version)
}

func runUpdate(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	log.Debug("Current module is '%s'.\n", moduleRoot)

	moduleFile := module.ReadModuleFile(moduleRoot)
	name := args[0]
	checkName(name)

	for idx, dep := range moduleFile.Dependencies {
		if dep.Name != name {
			continue
		}
		if url != "" {
			checkUrl(url)
			moduleFile.Dependencies[idx].URL = url
		}
		if version != "" {
			checkRev(version)
			moduleFile.Dependencies[idx].Version = module.Version{Rev: version}
		}
		module.WriteModuleFile(moduleRoot, moduleFile)
		log.Success("Updated dependency '%s'.\n", name)
		return
	}

	log.Warning("Module has no dependency '%s'.\n", name)
}

func runRemove(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	log.Debug("Current module is '%s'.\n", moduleRoot)

	moduleFile := module.ReadModuleFile(moduleRoot)
	name := args[0]
	checkName(name)

	for idx, dep := range moduleFile.Dependencies {
		if dep.Name != name {
			continue
		}
		moduleFile.Dependencies = append(moduleFile.Dependencies[:idx], moduleFile.Dependencies[idx+1:]...)
		module.WriteModuleFile(moduleRoot, moduleFile)
		log.Success("Removed dependency '%s'.\n", name)
		return
	}

	log.Warning("Module has no dependency on module '%s'.\n", name)
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

func checkRev(rev string) {
	if !revRegexp.MatchString(rev) {
		log.Fatal("Version '%s' does not match the expected format.\n", rev)
	}
}

func completeDepArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	completions := []string{}
	if len(args) == 0 {
		moduleRoot := util.GetModuleRoot()
		moduleFile := module.ReadModuleFile(moduleRoot)
		for _, dep := range moduleFile.Dependencies {
			completions = append(completions, dep.Name)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
