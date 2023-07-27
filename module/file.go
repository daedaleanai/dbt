package module

import (
	"path"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

type moduleFileVersion struct {
	Version uint
}

// MODULE file version 3 (current)

type Dependency struct {
	URL     string
	Version string
	Hash    string
	Type    string
}

type ModuleFile struct {
	Version      uint
	Layout       string
	Dependencies map[string]Dependency
	Overrides    map[string]Dependency
	Flags        map[string]string
}

// MODULE file version 2

type v2Dependency struct {
	URL     string
	Version string
}

type v2PinnedDependency struct {
	URL     string
	Version string
	Hash    string
}

type v2ModuleFile struct {
	Version            uint
	Layout             string
	Dependencies       map[string]v2Dependency
	PinnedDependencies map[string]v2PinnedDependency
	Flags              map[string]string
}

// MODULE file version 1

type v1Version struct {
	Rev  string
	Hash string
}

type v1Dependency struct {
	Name    string
	URL     string
	Version v1Version
}

type v1ModuleFile struct {
	Dependencies []v1Dependency
}

// ReadModuleFile reads and parses module Dependencies from a MODULE file.
func ReadModuleFile(modulePath string) ModuleFile {
	moduleFilePath := path.Join(modulePath, util.ModuleFileName)
	if !util.FileExists(moduleFilePath) {
		log.Debug("Module has no %s file.\n", util.ModuleFileName)
		return ModuleFile{
			Version:      util.DbtVersion[1],
			Dependencies: map[string]Dependency{},
		}
	}

	// Check MODULE file version.
	var moduleFileVersion moduleFileVersion
	util.ReadYaml(moduleFilePath, &moduleFileVersion)

	switch moduleFileVersion.Version {
	case 1:
		return readV1ModuleFile(moduleFilePath)
	case 2:
		return readV2ModuleFile(moduleFilePath)
	case util.DbtVersion[1]:
		return readV3ModuleFile(moduleFilePath)
	default:
		log.Fatal("MODULE file has version %d that requires a newer version of dbt\n", moduleFileVersion.Version)
		return ModuleFile{}
	}
}

// WriteModuleFile serializes and writes a Module's Dependencies to a MODULE file.
func WriteModuleFile(modulePath string, moduleFile ModuleFile) {
	moduleFile.Version = util.DbtVersion[1]
	moduleFilePath := path.Join(modulePath, util.ModuleFileName)
	util.WriteYaml(moduleFilePath, moduleFile)
}

func readV1ModuleFile(path string) ModuleFile {
	var v1ModuleFile v1ModuleFile
	util.ReadYaml(path, &v1ModuleFile)

	moduleFile := ModuleFile{
		Version:      util.DbtVersion[1],
		Dependencies: map[string]Dependency{},
	}
	for _, dep := range v1ModuleFile.Dependencies {
		moduleFile.Dependencies[dep.Name] = Dependency{
			URL:     dep.URL,
			Version: dep.Version.Rev,
			Hash:    dep.Version.Hash,
		}
	}
	return moduleFile
}

func readV2ModuleFile(path string) ModuleFile {
	var v2ModuleFile v2ModuleFile
	util.ReadYaml(path, &v2ModuleFile)

	moduleFile := ModuleFile{
		Version:      util.DbtVersion[1],
		Layout:       v2ModuleFile.Layout,
		Dependencies: map[string]Dependency{},
	}
	for name, dep := range v2ModuleFile.Dependencies {
		moduleFile.Dependencies[name] = Dependency{
			URL:     dep.URL,
			Version: dep.Version,
			Hash:    v2ModuleFile.PinnedDependencies[name].Hash,
		}
	}
	return moduleFile
}

func readV3ModuleFile(path string) ModuleFile {
	var moduleFile ModuleFile
	util.ReadYaml(path, &moduleFile)

	// YAML decoding can produce `nil`` maps if the key is present in the YAML file
	// but has no entries.
	if moduleFile.Dependencies == nil {
		moduleFile.Dependencies = map[string]Dependency{}
	}
	return moduleFile
}
