package module

import (
	"dwm/util"
	"io/ioutil"
	"path"
	"regexp"

	"github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
)

// Module represents a checked-out module.
type Module interface {
	Path() string
	Name() string
	IsDirty() (bool, error)
	HasRemote(string) (bool, error)
	HasVersionCheckedOut(version string) (bool, error)
	CheckoutVersion(version string) error
}

// OpenModule opens a module checked out on disk.
func OpenModule(modulePath string) (Module, error) {
	if util.FileExists(path.Join(modulePath, tarOriginFileName)) {
		return TarModule{modulePath}, nil
	}
	repo, err := git.PlainOpen(modulePath)
	if err != nil {
		return nil, err
	}
	return GitModule{modulePath, repo}, nil
}

var dependencyURLRegexp = regexp.MustCompile(`/([A-Za-z0-9_\-.]+)(\.git|\.tar\.gz)$`)

type moduleFile struct {
	Dependencies map[string]string
}

// Dependency represents a dependency one module has on another module.
type Dependency struct {
	URL     string
	Version string
}

// ModuleName parses the module name from the Dependency's URL.
func (d Dependency) ModuleName() string {
	return dependencyURLRegexp.FindStringSubmatch(d.URL)[1]
}

// ReadModuleFile reads and parses module Dependencies from a MODULE file.
func ReadModuleFile(moduleFilePath string) ([]Dependency, error) {
	data, err := ioutil.ReadFile(moduleFilePath)
	if err != nil {
		return nil, err
	}

	var moduleFile moduleFile
	err = yaml.Unmarshal(data, &moduleFile)
	if err != nil {
		return nil, err
	}

	deps := []Dependency{}
	for url, version := range moduleFile.Dependencies {
		deps = append(deps, Dependency{url, version})
	}

	return deps, nil
}

// WriteModuleFile serializes and writes a Module's Dependencies to a MODULE file.
func WriteModuleFile(moduleFilePath string, deps []Dependency) error {
	moduleFile := moduleFile{map[string]string{}}
	for _, dep := range deps {
		moduleFile.Dependencies[dep.URL] = dep.Version
	}

	data, err := yaml.Marshal(moduleFile)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(moduleFilePath, data, util.FileMode)
}
