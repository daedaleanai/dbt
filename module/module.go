package module

import (
	"bytes"
	"dwm/log"
	"dwm/util"
	"io/ioutil"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
)

const setupFileName = "SETUP.go"
const setupSentinelFileName = ".setup"

// Module represents a checked-out module.
type Module interface {
	Path() string
	Name() string
	IsDirty() bool
	HasRemote(string) bool
	HasVersionCheckedOut(version string) bool
	CheckoutVersion(version string)
}

// OpenModule opens a module checked out on disk.
func OpenModule(modulePath string) Module {
	log.Debug("Opening module '%s'.\n", modulePath)

	if util.DirExists(path.Join(modulePath, ".git")) {
		log.Debug("Found '.git' directory. Expecting this to be a GitModule.\n")
		repo, err := git.PlainOpen(modulePath)
		if err != nil {
			log.Error("Failed to open repo: %s.\n", modulePath, err)
		}
		return GitModule{modulePath, repo}
	}

	if util.FileExists(path.Join(modulePath, tarOriginFileName)) {
		log.Debug("Found '.origin' file. Expecting this to be a TarModule.\n")
		return TarModule{modulePath}
	}

	log.Error("Failed to open module '%s': Could not determine module type. Try to remove the module directory and rerun the command.\n", modulePath)
	return nil
}

// OpenOrCreateModule tries to open the module in `modulePath`. If the `modulePath` directory does
// not yet exists, it tries to create a new module by cloning / downloading the module from `url`.
func OpenOrCreateModule(modulePath string, url string) Module {
	log.Debug("Opening or creating module '%s' from url '%s'.\n", modulePath, url)
	if util.DirExists(modulePath) {
		log.Debug("Module directory exists.\n")
		return OpenModule(modulePath)
	}

	log.Debug("Module directory does not exists.\n")
	if strings.HasSuffix(url, ".git") {
		log.Debug("Module URL ends in '.git'. Trying to create a new GitModule.\n")
		return CreateGitModule(modulePath, url)
	}
	if strings.HasSuffix(url, ".tar.gz") {
		log.Debug("Module URL ends in '.tar.gz'. Trying to create a new TarModule.\n")
		return CreateTarModule(modulePath, url)
	}

	log.Error("Failed to determine module type from dependency url '%s'.\n", url)
	return nil
}

// SetupModule runs the SETUP.go file in the root directory of `mod` (it if exists).
func SetupModule(mod Module) {
	log.Debug("Trying to set up module '%s'.\n", mod.Name())

	setupSentinelFilePath := path.Join(mod.Path(), setupSentinelFileName)
	if util.FileExists(setupSentinelFilePath) {
		log.Debug("'.setup' file already exists in module directory. Nothing to do.\n")
		return
	}

	setupFilePath := path.Join(mod.Path(), setupFileName)
	if !util.FileExists(setupFilePath) {
		log.Debug("Module has no SETUP.go file. Nothing to do.\n")
		return
	}

	log.Log("Running 'go run %s'.\n", setupFilePath)
	log.Spinner.Start()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", setupFilePath)
	cmd.Dir = mod.Path()
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	log.Spinner.Stop()
	if err != nil {
		log.Error("Running SETUP.go failed:\nSTDOUT:\n%s\nSTDERR:\n%s", string(stdout.Bytes()), string(stderr.Bytes()))
	}
	log.Success("Module is set up.\n")

	log.Debug("Creating sentinel file.\n", mod.Name())
	err = ioutil.WriteFile(setupSentinelFilePath, []byte{}, util.FileMode)
	if err != nil {
		log.Error("Could not create .setup sentinel file: %s\n.", err)
	}
}

// GetModuleDependencies returns the Dependencies of a module by reading its MODULE file.
func GetModuleDependencies(modulePath string) []Dependency {
	moduleFilePath := path.Join(modulePath, util.ModuleFileName)
	if !util.FileExists(moduleFilePath) {
		log.Warning("Module has no MODULE file.\n")
		return []Dependency{}
	}
	deps, err := ReadModuleFile(moduleFilePath)
	if err != nil {
		log.Error("Failed to read MODULE file: %s.\n", err)
	}
	return deps
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
