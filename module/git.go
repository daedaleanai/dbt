package module

import (
	"bytes"
	"os/exec"
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
func createGitModule(modulePath, url string) (Module, error) {
	mod := GitModule{modulePath}
	util.MkdirAll(modulePath)
	log.Log("Cloning '%s'.\n", url)
	if _, _, err := mod.tryRunGitCommand("clone", "--recursive", url, modulePath); err != nil {
		return nil, err
	}
	return mod, nil
}

func (m GitModule) RootPath() string {
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
	return string(m.runGitCommand("rev-list", "-n", "1", ref))
}

// IsDirty returns whether the underlying repository has any uncommited changes.
func (m GitModule) IsDirty() bool {
	return len(m.runGitCommand("status", "-s")) > 0
}

// IsAncestor returns whether ancestor is an ancestor of rev in the commit tree.
func (m GitModule) IsAncestor(ancestor, rev string) bool {
	_, _, err := m.tryRunGitCommand("merge-base", "--is-ancestor", ancestor, rev)
	return err == nil
}

// Fetch fetches changes from the default remote and reports whether any updates have been fetched.
func (m GitModule) Fetch() bool {
	if m.IsDirty() {
		// If the module has uncommited changes, it does not match any version.
		log.Warning("The module has uncommited changes. Not fetching any changes.\n")
		return false
	}

	return len(m.runGitCommand("fetch", "--all", "--tags")) > 0
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
	stdout, stderr, err := m.tryRunGitCommand(args...)
	if err != nil {
		log.Fatal("Failed to run git command 'git %s':\n%s\n%s\n%s\n", strings.Join(args, " "), stderr, stdout, err)
	}
	return stdout
}

func (m GitModule) tryRunGitCommand(args ...string) (string, string, error) {
	stderr := bytes.Buffer{}
	stdout := bytes.Buffer{}
	log.Debug("Running git command: git %s\n", strings.Join(args, " "))
	cmd := exec.Command("git", args...)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	cmd.Dir = m.path
	err := cmd.Run()
	return strings.TrimSuffix(stdout.String(), "\n"), strings.TrimSuffix(stderr.String(), "\n"), err
}
