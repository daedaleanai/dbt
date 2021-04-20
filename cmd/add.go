package cmd

import (
	"regexp"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"

	"github.com/daedaleanai/cobra"
)

var depNameAndVersionRegexp = regexp.MustCompile(`^[A-Za-z0-9_\-.]+$`)
var depUrlRegexp = regexp.MustCompile(`/([A-Za-z0-9_\-.]+)(\.git|\.tar\.gz)$`)

var addCmd = &cobra.Command{
	Use:   "add [NAME] Url VERSION",
	Args:  cobra.RangeArgs(2, 3),
	Short: "Adds a dependency to the MODULE file of the current module",
	Long: `Adds a dependency to the MODULE file of the current module.
If the MODULE file already has an entry for the dependency, the version of the existing entry is updated.`,
	Run: runAdd,
}

func init() {
	depCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) {
	moduleRoot := util.GetModuleRoot()
	log.Log("Current module is '%s'.\n", moduleRoot)

	moduleFile := module.ReadModuleFile(moduleRoot)

	var newDep module.Dependency
	if len(args) == 2 {
		newDep = module.Dependency{
			Name:    parseNameFromUrl(args[0]),
			URL:     args[0],
			Version: module.Version{Rev: args[1], Hash: ""},
		}
	} else {
		newDep = module.Dependency{
			Name:    args[0],
			URL:     args[1],
			Version: module.Version{Rev: args[2], Hash: ""},
		}
	}

	if !depNameAndVersionRegexp.MatchString(newDep.Name) {
		log.Fatal("Dependency name '%s' contains unallowed characters.\n", newDep.Name)
	}
	if !depNameAndVersionRegexp.MatchString(newDep.Version.Rev) {
		log.Fatal("Dependency version '%s' contains unallowed characters.\n", newDep.Version.Rev)
	}
	if !depUrlRegexp.MatchString(newDep.URL) {
		log.Fatal("Dependency url '%s' contains does not match the expected format.\n", newDep.URL)
	}

	for idx, dep := range moduleFile.Dependencies {
		if dep.URL != newDep.URL {
			continue
		}
		if dep.Name != newDep.Name {
			log.Fatal("Module already has a dependency for that Url with a different name.\n")
		}
		if dep.Version != newDep.Version {
			moduleFile.Dependencies[idx] = newDep
			module.WriteModuleFile(moduleRoot, moduleFile)
			log.Success("Updated version of dependency on module '%s' to '%s'.\n", newDep.Name, newDep.Version.Rev)
			return
		}
		log.Success("Module already depends on module '%s', version '%s'.\n", newDep.Name, newDep.Version.Rev)
		return
	}

	moduleFile.Dependencies = append(moduleFile.Dependencies, newDep)
	module.WriteModuleFile(moduleRoot, moduleFile)
	log.Success("Added dependency on module '%s', version '%s'.\n", newDep.Name, newDep.Version.Rev)
}

func parseNameFromUrl(url string) string {
	match := depUrlRegexp.FindStringSubmatch(url)
	if len(match) < 2 {
		log.Fatal("Dependency url '%s' contains does not match the expected format.\n", url)
	}
	return match[1]
}
