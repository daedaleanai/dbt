package module

import (
	"dbt/log"
	"path"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GitModule is a module backed by a git repository.
type GitModule struct {
	path string
	repo *git.Repository
}

// CreateGitModule creates a new GitModule in the given `modulePath`
// by cloning the repository from `url`.
func CreateGitModule(modulePath, url string) Module {
	log.Log("Cloning '%s'.\n", url)
	log.Spinner.Start()
	repo, err := git.PlainClone(modulePath, false, &git.CloneOptions{
		URL: url,
	})
	log.Spinner.Stop()
	if err != nil {
		log.Fatal("Failed to clone GitModule: %s.\n", err)
	}
	mod := GitModule{modulePath, repo}
	SetupModule(mod)
	return mod
}

// Path returns the on-disk path of the module.
func (m GitModule) Path() string {
	return m.path
}

// Name returns the name of the module.
func (m GitModule) Name() string {
	return path.Base(m.Path())
}

// IsDirty returns whether the underlying repository has any uncommited changes.
func (m GitModule) IsDirty() bool {
	worktree, err := m.repo.Worktree()
	if err != nil {
		log.Fatal("Failed to get repo worktree: %s.\n", err)
	}
	status, err := worktree.Status()
	if err != nil {
		log.Fatal("Failed to get repo status: %s.\n", err)
	}
	return !status.IsClean()
}

// HasOrigin returns whether the underlying repository has a remote called origin that matches `url`.
func (m GitModule) HasOrigin(url string) bool {
<<<<<<< HEAD
	for _, originURL := range m.origin().Config().URLs {
		if originURL == url {
			return true
=======
	remotes, err := m.repo.Remotes()
	if err != nil {
		log.Fatal("Failed to get repo remotes: %s.\n", err)
	}
	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			for _, remoteURL := range remote.Config().URLs {
				if remoteURL == url {
					return true
				}
			}
>>>>>>> added status command
		}
	}
	return false
}

// HasVersionCheckedOut returns whether the current module's version matches `version`.
func (m GitModule) HasVersionCheckedOut(version string) bool {
	if m.IsDirty() {
		// If the module has uncommited changes, it does not match any version.
		log.Debug("The module has uncommited changes.\n")
		return false
	}

	// Convert the version (which might be a hash, tag, branch, etc.) to a cannonical commit hash
	// before comparing it to HEAD.
	hash, err := m.repo.ResolveRevision(plumbing.Revision(version))
	if err != nil {
		log.Debug("Failed to resolve revision '%s': %s.\n", version, err)
		return false
	}
	log.Debug("Version '%s' was resolved to commit hash '%s'.\n", version, hash.String())

	head, err := m.repo.Head()
	if err != nil {
		log.Fatal("Failed to get repo HEAD: %s.\n", err)
	}
	log.Debug("Repo HEAD is '%s'.\n", head.Hash().String())

	return head.Hash() == *hash
}

// CheckoutVersion changes the current module's version to `version`.
func (m GitModule) CheckoutVersion(version string) {
	worktree, err := m.repo.Worktree()
	if err != nil {
		log.Fatal("Failed to get repo worktree: %s.\n", err)
	}

	// Convert the version (which might be a hash, tag, branch, etc.) to a cannonical commit hash.
	hash, err := m.repo.ResolveRevision(plumbing.Revision(version))
	if err != nil {
		log.Fatal("Failed to resolve revision '%s': %s.\n", version, err)
	}
	log.Debug("Version '%s' was resolved to commit hash '%s'.\n", version, hash.String())

	err = worktree.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	})
	if err != nil {
		log.Fatal("Failed to checkout version '%s': %s.\n", hash.String(), err)
	}
}

// Fetch fetches changes from the 'origin' remote and reports whether any updates have bee fetched.
func (m GitModule) Fetch() bool {
	if m.IsDirty() {
		// If the module has uncommited changes, it does not match any version.
		log.Warning("The module has uncommited changes. Not pulling any changes.\n")
		return false
	}

	origin := m.origin()
	log.Spinner.Start()
	err := origin.Fetch(&git.FetchOptions{
		Tags: git.AllTags,
	})
	log.Spinner.Stop()

	if err == nil {
		return true
	}
	if err != git.NoErrAlreadyUpToDate {
		log.Fatal("Failed to fetch changes: %s.\n", err)
	}
	return false
}

func (m GitModule) origin() *git.Remote {
	remotes, err := m.repo.Remotes()
	if err != nil {
		log.Fatal("Failed to get repository remotes: %s.\n", err)
	}
	for _, remote := range remotes {
		if remote.Config().Name == "origin" {
			return remote
		}
	}

	log.Fatal("Failed to get 'origin' remote: repository has no such remote.\n")
	return nil
}

// CheckedOutVersions returns all currently checked out versions.
// This includes the HEAD commit hash and all annotated tags that point to HEAD.
func (m GitModule) CheckedOutVersions() []string {
	if m.IsDirty() {
		return []string{}
	}

	head := m.head().Hash()
	versions := []string{head.String()}

	tags, err := m.repo.TagObjects()
	if err != nil {
		log.Fatal("Failed to read tags: %s.\n", err)
	}
	tags.ForEach(func(tag *object.Tag) error {
		if tag.Target == head {
			versions = append(versions, tag.Name)
		}
		return nil
	})
	return versions
}

func (m GitModule) head() *plumbing.Reference {
	head, err := m.repo.Head()
	if err != nil {
		log.Fatal("Failed to get repo HEAD: %s.\n", err)
	}
	return head
}

// CheckedOutVersions returns all currently checked out versions.
// This includes the HEAD commit hash and all annotated tags the point to HEAD.
func (m GitModule) CheckedOutVersions() []string {
	if m.IsDirty() {
		return []string{}
	}

	head := m.head().Hash()
	versions := []string{head.String()}

	tags, err := m.repo.TagObjects()
	if err != nil {

	}
	err = tags.ForEach(func(tag *object.Tag) error {
		if tag.Target == head {
			versions = append(versions, tag.Name)
		}
		return nil
	})
	return versions
}

func (m GitModule) head() *plumbing.Reference {
	head, err := m.repo.Head()
	if err != nil {
		log.Fatal("Failed to get repo HEAD: %s.\n", err)
	}
	return head
}
