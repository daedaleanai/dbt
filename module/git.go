package module

import (
	"path"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GitModule is a module backed by a git repository.
type GitModule struct {
	path string
	repo *git.Repository
}

// CreateGitModule creates a new GitModule in the given `modulePath`
// by cloning the repository from `url`.
func CreateGitModule(modulePath, url string) (Module, error) {
	repo, err := git.PlainClone(modulePath, false, &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return nil, err
	}
	return GitModule{modulePath, repo}, nil
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
func (m GitModule) IsDirty() (bool, error) {
	worktree, err := m.repo.Worktree()
	if err != nil {
		return false, err
	}
	status, err := worktree.Status()
	if err != nil {
		return false, err
	}
	return !status.IsClean(), nil
}

// HasRemote returns whether the underlying repository has a remote matching `url`.
func (m GitModule) HasRemote(url string) (bool, error) {
	remotes, err := m.repo.Remotes()
	if err != nil {
		return false, err
	}
	for _, remote := range remotes {
		for _, remoteURL := range remote.Config().URLs {
			if remoteURL == url {
				return true, nil
			}
		}
	}

	return false, nil
}

// HasVersionCheckedOut returns whether the current module's version matches `version`.
func (m GitModule) HasVersionCheckedOut(version string) (bool, error) {
	isDirty, err := m.IsDirty()
	if err != nil {
		return false, err
	}

	if isDirty {
		// If the module has uncommited changes, it does not match any version.
		return false, nil
	}

	// Convert the version (which might be a hash, tag, branch, etc.) to a cannonical commit hash
	// before comparing it to HEAD.
	versionHash, err := m.repo.ResolveRevision(plumbing.Revision(version))
	if err != nil {
		return false, err
	}

	head, err := m.repo.Head()
	if err != nil {
		return false, err
	}

	return (head.Hash() == *versionHash), nil
}

// CheckoutVersion changes the current module's version to `version`.
func (m GitModule) CheckoutVersion(version string) error {
	worktree, err := m.repo.Worktree()
	if err != nil {
		return err
	}

	// Convert the version (which might be a hash, tag, branch, etc.) to a cannonical commit hash.
	hash, err := m.repo.ResolveRevision(plumbing.Revision(version))
	if err != nil {
		return err
	}

	return worktree.Checkout(&git.CheckoutOptions{
		Hash: *hash,
	})
}
