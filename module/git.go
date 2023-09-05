package module

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/daedaleanai/dbt/config"
	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"
)

// GitModule is a module backed by a git repository.
type GitModule struct {
	path   string
	mirror *GitMirror
}

// GitMirror is a bare repository that backs a GitModule
type GitMirror struct {
	path string
}

// Obtains a mirror for a git repository if the global mirror directory has been set up
func getOrCreateGitMirror(url string) (*GitMirror, error) {
	configuration := config.GetConfig()
	if configuration.Mirror == "" {
		log.Debug("Mirrors are not configured.\n")
		return nil, nil
	}

	urlHash := sha256.Sum256([]byte(url))
	urlHashString := fmt.Sprintf("git-%x", urlHash[:])
	mirrorPath := path.Join(configuration.Mirror, urlHashString)

	log.Debug("Looking for mirror of '%s' in directory '%s'.\n", url, mirrorPath)

	if util.DirExists(mirrorPath) {
		log.Debug("Mirror found at '%s'.\n", mirrorPath)
		return &GitMirror{path: mirrorPath}, nil
	}

	util.MkdirAll(mirrorPath)
	mod := GitModule{mirrorPath, nil}
	if err := mod.clone(url, true); err != nil {
		return nil, err
	}
	log.Debug("Mirror cloned at '%s'.\n", mirrorPath)

	return &GitMirror{path: mirrorPath}, nil
}

// createGitModule creates a new GitModule in the given `modulePath`
// by cloning the repository from `url`.
func CreateGitModule(modulePath, url string) (Module, error) {
	// Figure out if there is a local mirror for it
	mirror, err := getOrCreateGitMirror(url)
	if err != nil {
		return nil, err
	}

	mod := GitModule{modulePath, mirror}
	util.MkdirAll(modulePath)
	if err := mod.clone(url, false); err != nil {
		return nil, err
	}

	return mod, nil
}

func (m GitModule) Name() string {
	return strings.TrimSuffix(path.Base(m.URL()), ".git")
}

func (m GitModule) RootPath() string {
	return m.path
}

// URL returns the url of the underlying git repository.
func (m GitModule) URL() string {
	return m.runGitCommand("config", "--get", "remote.origin.url")
}

// Mirror returns the path of the mirror
func (m GitModule) Mirror() *GitMirror {
	return m.mirror
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

func (m GitModule) Type() ModuleType {
	return GitModuleType
}

// GetMergeBase returns the best common ancestor that could be used for a merge between the two given references.
func (m GitModule) GetMergeBase(revA, revB string) (string, error) {
	stdout, _, err := m.tryRunGitCommand("merge-base", revA, revB)
	return stdout, err
}

func (m GitModule) GetCommitTitle(revision string) (string, error) {
	stdout, _, err := m.tryRunGitCommand("show", "--format=format:\"%s\"", "-s", revision)
	return stdout, err
}

func (m GitModule) GetCommitAuthorName(revision string) (string, error) {
	stdout, _, err := m.tryRunGitCommand("show", "--format=format:\"%an\"", "-s", revision)
	return stdout, err
}

func (m GitModule) GetCommitsBetweenRefs(base, head string) ([]string, error) {
	result := []string{}
	stdout, _, err := m.tryRunGitCommand("rev-list", strings.Join([]string{strings.TrimSpace(base), strings.TrimSpace(head)}, ".."))
	for _, line := range strings.Split(stdout, "\n") {
		trimmedLine := strings.TrimSpace(line)
		if len(trimmedLine) == 0 {
			continue
		}

		result = append(result, trimmedLine)
	}
	return result, err
}

// Runs a git command with the specified arguments, exiting with an error message if the command
// could not be executed
func (m GitModule) runGitCommand(args ...string) string {
	stdout, stderr, err := m.tryRunGitCommand(args...)
	if err != nil {
		log.Fatal("Failed to run git command 'git %s':\n%s\n%s\n%s\n", strings.Join(args, " "), stderr, stdout, err)
	}
	return stdout
}

// Tries to run a git subcommand and return stdout, stderr and an error if the process exited with
// an exit code != 0
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

// Clones a module from the given url at the specfied path location. If asMirror is passed, then a
// mirror is created instead of a regular git repository.
// If the git module has a mirror assigned, it will be used as the reference for the new git repository.
func (m GitModule) clone(url string, asMirror bool) error {
	var err error
	if asMirror {
		log.Debug("Cloning '%s' as mirror '%s'.\n", url, m.path)
		_, _, err = m.tryRunGitCommand("clone", "--mirror", url, m.path)
	} else if m.mirror != nil {
		log.Log("Cloning '%s' using mirror '%s'.\n", url, m.mirror.path)
		_, _, err = m.tryRunGitCommand("clone", "--recursive", "--reference", m.mirror.path, url, m.path)
	} else {
		log.Log("Cloning '%s'.\n", url)
		_, _, err = m.tryRunGitCommand("clone", "--recursive", url, m.path)
	}
	if err != nil {
		// Leave clean state so that the operation can be retried
		util.RemoveDir(m.path)
	}
	return err
}
