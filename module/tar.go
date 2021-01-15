package module

import (
	"archive/tar"
	"compress/gzip"
	"dwm/util"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
)

const tarOriginFileName = ".origin"
const tarDefaultVersion = "static"

// TarModule is a module backed by a tar.gz archive.
// TarModules only have a single "static" version.
type TarModule struct {
	path string
}

// CreateTarModule creates a new TarModule in the given `modulePath` by downloading
// and extracting the TAR archive reference by `url`. The origin of the module
// (i.e., the download url) is stored in a ".origin" file inside the module directory.
func CreateTarModule(modulePath, url string) (Module, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	tarFile, err := gzip.NewReader(response.Body)
	if err != nil {
		return nil, err
	}

	tarReader := tar.NewReader(tarFile)
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			dirPath := path.Join(modulePath, header.Name)
			err := os.MkdirAll(dirPath, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}
		case tar.TypeReg:
			filePath := path.Join(modulePath, header.Name)
			file, err := os.Create(filePath)
			if err != nil {
				return nil, err
			}
			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				return nil, err
			}
			err = os.Chmod(filePath, os.FileMode(header.Mode))
			if err != nil {
				return nil, err
			}
		case tar.TypeLink:
			oldname := path.Join(modulePath, header.Linkname)
			newname := path.Join(modulePath, header.Name)
			err = os.Link(oldname, newname)
			if err != nil {
				return nil, err
			}
		case tar.TypeSymlink:
			oldname := path.Join(modulePath, header.Linkname)
			newname := path.Join(modulePath, header.Name)
			err = os.Symlink(oldname, newname)
			if err != nil {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("unknown tar type flag: %d in %s", header.Typeflag, header.Name)
		}
	}

	err = ioutil.WriteFile(path.Join(modulePath, tarOriginFileName), []byte(url), util.FileMode)
	if err != nil {
		return nil, err
	}

	return OpenModule(modulePath)
}

// Path returns the on-disk path of the module.
func (m TarModule) Path() string {
	return m.path
}

// Name returns the name of the module.
func (m TarModule) Name() string {
	return path.Base(m.path)
}

// IsDirty returns whether the module has any uncommited changes.
// TarModules never have any uncommited changes by definition.
func (m TarModule) IsDirty() (bool, error) {
	return false, nil
}

// HasRemote returns whether the origin of the module matches `url` by
// checking the ".origin" file inside the module directory.
func (m TarModule) HasRemote(url string) (bool, error) {
	data, err := ioutil.ReadFile(path.Join(m.path, tarOriginFileName))
	if err != nil {
		return false, err
	}
	return (string(data) == url), nil
}

// HasVersionCheckedOut returns whether the module's current version matched `version`.
// However, TarModules only have a single "static" version.
func (m TarModule) HasVersionCheckedOut(version string) (bool, error) {
	return (version == tarDefaultVersion), nil
}

// CheckoutVersion changes the module's current version to `version` if possible.
// However, TarModules only have a single "static" version. Attempting to check out any
// other version results in an error.
func (m TarModule) CheckoutVersion(version string) error {
	if version != tarDefaultVersion {
		return fmt.Errorf("cannot change version of tar module")
	}
	return nil
}
