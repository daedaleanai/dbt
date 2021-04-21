package module

import (
	"path"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

type Version struct {
	Rev  string
	Hash string
}

type Dependency struct {
	Name    string
	URL     string
	Version Version
}

type ModuleFile struct {
	Dependencies []Dependency
}

// ReadModuleFile reads and parses module Dependencies from a MODULE file.
func ReadModuleFile(modulePath string) ModuleFile {
	var moduleFile ModuleFile

	moduleFilePath := path.Join(modulePath, util.ModuleFileName)
	if !util.FileExists(moduleFilePath) {
		log.Debug("Module has no '%s' file.\n", util.ModuleFileName)
		return moduleFile
	}

	util.ReadYaml(moduleFilePath, &moduleFile)

	// Check that there are no duplicate names.
	names := map[string]bool{}
	for _, dep := range moduleFile.Dependencies {
		if _, exists := names[dep.Name]; exists {
			log.Fatal("%s file '%s' contains duplicate dependency name '%s'. Please clean up the file manually and try again.\n", util.ModuleFileName, moduleFilePath, dep.Name)
		}
		names[dep.Name] = true
	}

	return moduleFile
}

// WriteModuleFile serializes and writes a Module's Dependencies to a MODULE file.
func WriteModuleFile(modulePath string, moduleFile ModuleFile) {
	moduleFilePath := path.Join(modulePath, util.ModuleFileName)
	util.WriteYaml(moduleFilePath, moduleFile)
}
