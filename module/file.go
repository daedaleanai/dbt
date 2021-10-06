package module

import (
	"path"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

type Dependency struct {
	URL     string
	Version string
}

type PinnedDependency struct {
	URL     string
	Version string
	Hash    string
}

type ModuleFile struct {
	Version            uint
	Dependencies       map[string]Dependency
	PinnedDependencies map[string]PinnedDependency
}

type moduleFileVersion struct {
	Version uint
}

type legacyVersion struct {
	Rev  string
	Hash string
}

type legacyDependency struct {
	Name    string
	URL     string
	Version legacyVersion
}

type legacyModuleFile struct {
	Dependencies []legacyDependency
}

// ReadModuleFile reads and parses module Dependencies from a MODULE file.
func ReadModuleFile(modulePath string) ModuleFile {
	moduleFile := ModuleFile{
		Version:            util.DbtVersion[1],
		Dependencies:       map[string]Dependency{},
		PinnedDependencies: map[string]PinnedDependency{},
	}

	moduleFilePath := path.Join(modulePath, util.ModuleFileName)
	if !util.FileExists(moduleFilePath) {
		log.Debug("Module has no %s file.\n", util.ModuleFileName)
		return moduleFile
	}

	// Check MODULE file version.
	var moduleFileVersion moduleFileVersion
	util.ReadYaml(moduleFilePath, &moduleFileVersion)
	if moduleFileVersion.Version == util.DbtVersion[1] {
		util.ReadYaml(moduleFilePath, &moduleFile)
	} else {
		var legacyModuleFile legacyModuleFile
		util.ReadYaml(moduleFilePath, &legacyModuleFile)
		for _, legacyDep := range legacyModuleFile.Dependencies {
			dep := Dependency{URL: legacyDep.URL, Version: legacyDep.Version.Rev}
			moduleFile.Dependencies[legacyDep.Name] = dep
			if legacyDep.Version.Hash != "" {
				moduleFile.PinnedDependencies[legacyDep.Name] = PinnedDependency{
					URL:     dep.URL,
					Version: dep.Version,
					Hash:    legacyDep.Version.Hash,
				}
			}
		}
	}
	// YAML decoding can nil maps if the key is present in the yaml file
	// with no entries. Restore them if this happens.
	if moduleFile.Dependencies == nil {
		moduleFile.Dependencies = map[string]Dependency{}
	}
	if moduleFile.PinnedDependencies == nil {
		moduleFile.PinnedDependencies = map[string]PinnedDependency{}
	}
	return moduleFile
}

// WriteModuleFile serializes and writes a Module's Dependencies to a MODULE file.
func WriteModuleFile(modulePath string, moduleFile ModuleFile) {
	moduleFile.Version = util.DbtVersion[1]
	moduleFilePath := path.Join(modulePath, util.ModuleFileName)
	util.WriteYaml(moduleFilePath, moduleFile)
}
