package module

import (
	"path"

	"github.com/daedaleanai/dbt/log"
)

const localDefaultVersion = "master"

// LocalModule is a module that is not backed by a git repository or .tar.gz archive.
type LocalModule struct {
	path string
}

// Name returns the name of the module.
func (m LocalModule) Name() string {
	return path.Base(m.path)
}

// Path returns the on-disk path of the module.
func (m LocalModule) Path() string {
	return m.path
}

// URL must never be called on a LocalModule.
func (m LocalModule) URL() string {
	log.Fatal("URL() must never be called on a LocalModule.\n")
	return ""
}

// Head returns the default version for all LocalModules.
func (m LocalModule) Head() string {
	return localDefaultVersion
}

// RevParse must never be called on a LocalModule.
func (m LocalModule) RevParse(ref string) string {
	log.Fatal("RevParse() must never be called on a LocalModule.\n")
	return ""
}

// IsDirty always returns false, because LocalModules are never dirty by definition.
func (m LocalModule) IsDirty() bool {
	return false
}

// Fetch does nothing on LocalModules and reports that no changes have been fetched.
func (m LocalModule) Fetch() bool {
	return false
}

// Checkout must never be called on a LocalModule.
func (m LocalModule) Checkout(ref string) {
	log.Fatal("Checkout() must never be called on a LocalModule.\n")
}
