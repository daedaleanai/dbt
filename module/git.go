package module

import (
	"bytes"
	"os/exec"
	"path"
	"strings"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

// GitModule is a module backed by a git repository.
type GitModule struct {
	path string
}

// createGitModule creates a new GitModule in the given `modulePath`
// by cloning the repository from `url`.
func createGitModule(modulePath, url string) Module {
	mod := GitModule{modulePath}
	util.MkdirAll(modulePath)
	log.Log("Cloning '%s'.\n", url)
	log.Spinner.Start()
	mod.runGitCommand("clone", url, modulePath)
	log.Spinner.Stop()
	SetupModule(mod)
	return mod
}

// Name returns the name of the module.
func (m GitModule) Name() string {
	return path.Base(m.Path())
}

// Path returns the on-disk path of the module.
func (m GitModule) Path() string {
	return m.path
}

// URL returns the url of the underlying git repository.
func (m GitModule) URL() string {
	return m.runGitCommand("config", "--get", "remote.origin.url")
}

// Head returns the commit hash of the HEAD of the underlying git repository.
func (m GitModule) Head() string {
	return m.RevParse("HEAD")
}

// RevParse returns the commit hash for the commit referenced by `ref`.
func (m GitModule) RevParse(ref string) string {
	return string(m.runGitCommand("rev-parse", ref))
}

// IsDirty returns whether the underlying repository has any uncommited changes.
func (m GitModule) IsDirty() bool {
	return len(m.runGitCommand("status", "-s")) > 0
}

// Fetch fetches changes from the default remote and reports whether any updates have been fetched.
func (m GitModule) Fetch() bool {
	if m.IsDirty() {
		// If the module has uncommited changes, it does not match any version.
		log.Warning("The module has uncommited changes. Not fetching any changes.\n")
		return false
	}

	return len(m.runGitCommand("fetch")) > 0
}

// Checkout changes the current module's version to `ref`.
func (m GitModule) Checkout(ref string) {
	if m.IsDirty() {
		// If the module has uncommited changes, it does not match any version.
		log.Debug("The module has uncommited changes.\n")
		return
	}

	m.runGitCommand("checkout", ref)
}

func (m GitModule) runGitCommand(args ...string) string {
	stderr := bytes.Buffer{}
	stdout := bytes.Buffer{}
	log.Debug("Running git command: git %s\n", strings.Join(args, " "))
	log.Spinner.Start()
	cmd := exec.Command("git", args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Dir = m.path
	err := cmd.Run()
	log.Spinner.Stop()
	if err != nil {
		log.Fatal("Failed to run git command 'git %s':\n%s\n%s\n", strings.Join(args, " "), stderr.String(), err)
	}
	return strings.TrimSuffix(stdout.String(), "\n")
}
