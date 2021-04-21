package cmd

import (
	"regexp"

	"github.com/daedaleanai/cobra"
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/module"
	"github.com/daedaleanai/dbt/util"
)

var depNameAndVersionRegexp = regexp.MustCompile(`^[A-Za-z0-9_\-.]+$`)
var depUrlRegexp = regexp.MustCompile(`/([A-Za-z0-9_\-.]+)(\.git|\.tar\.gz)$`)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds a dependency to the MODULE file of the current module",
}

func init() {
	depCmd.AddCommand(addCmd)
}

func parseNameFromUrl(url string) string {
	match := depUrlRegexp.FindStringSubmatch(url)
	if len(match) < 2 {
		log.Fatal("Dependency url '%s' contains does not match the expected format.\n", url)
	}
	return match[1]
}

func addDependency(newDep module.Dependency) {
	moduleRoot := util.GetModuleRoot()
	log.Log("Current module is '%s'.\n", moduleRoot)
	moduleFile := module.ReadModuleFile(moduleRoot)

	if !depNameAndVersionRegexp.MatchString(newDep.Name) {
		log.Fatal("Dependency name '%s' contains forbidden characters.\n", newDep.Name)
	}
	if !depNameAndVersionRegexp.MatchString(newDep.Version.Rev) {
		log.Fatal("Dependency version '%s' contains forbidden characters.\n", newDep.Version.Rev)
	}
	if !depUrlRegexp.MatchString(newDep.URL) {
		log.Fatal("Dependency url '%s' contains does not match the expected format.\n", newDep.URL)
	}

	for idx, dep := range moduleFile.Dependencies {
		if dep.URL != newDep.URL {
			continue
		}
		if dep.Name != newDep.Name {
			log.Fatal("Module already has a dependency for that URL with a different name.\n")
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
