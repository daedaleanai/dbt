package module

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/daedaleanai/dbt/log"
	"github.com/daedaleanai/dbt/util"

	"gopkg.in/yaml.v2"
)

const tarMetadataFileName = ".metadata"
const tarDefaultVersion = "master"

type metadataFile struct {
	Origin string
}

// TarModule is a module backed by a tar.gz archive.
// TarModules only have a single "master" version.
type TarModule struct {
	path string
}

// CreateTarModule creates a new TarModule in the given `modulePath` by downloading
// and extracting the TAR archive reference by `url`. The origin of the module
// (i.e., the download url) is stored in a ".metadata" file inside the module directory.
func CreateTarModule(modulePath, url string) Module {
	log.Log("Downloading '%s'.\n", url)
	log.Spinner.Start()
	defer log.Spinner.Stop()

	response, err := http.Get(url)
	if err != nil {
		log.Fatal("Failed to download archive: %s.\n", err)
	}
	defer response.Body.Close()

	tarFile, err := gzip.NewReader(response.Body)
	if err != nil {
		log.Fatal("Failed to decompress: %s.\n")
	}

	tarReader := tar.NewReader(tarFile)
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal("Failed to decompress: %s.\n")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			dirPath := path.Join(modulePath, header.Name)
			log.Debug("Creating directory '%s'.\n", dirPath)
			err := os.MkdirAll(dirPath, os.FileMode(header.Mode))
			if err != nil {
				log.Fatal("Failed to create directory while decompressing archive: %s.\n", err)
			}
		case tar.TypeReg:
			filePath := path.Join(modulePath, header.Name)
			log.Debug("Creating file '%s'.\n", filePath)
			file, err := os.Create(filePath)
			if err != nil {
				log.Fatal("Failed to create file while decompressing archive: %s.\n", err)
			}
			_, err = io.Copy(file, tarReader)
			file.Close()
			if err != nil {
				log.Fatal("Failed to writing file while decompressing archive: %s.\n", err)
			}
			err = os.Chmod(filePath, os.FileMode(header.Mode))
			if err != nil {
				log.Fatal("Failed to change filemode while decompressing archive: %s.\n", err)
			}
		case tar.TypeLink:
			oldname := path.Join(modulePath, header.Linkname)
			newname := path.Join(modulePath, header.Name)
			log.Debug("Creating link from '%s' to '%s'.\n", newname, oldname)
			err = os.Link(oldname, newname)
			if err != nil {
				log.Fatal("Failed to create link while decompressing archive: %s.\n", err)
			}
		case tar.TypeSymlink:
			oldname := path.Join(modulePath, header.Linkname)
			newname := path.Join(modulePath, header.Name)
			log.Debug("Creating symlink from '%s' to '%s'.\n", newname, oldname)
			err = os.Symlink(oldname, newname)
			if err != nil {
				log.Fatal("Failed to create symlink while decompressing archive: %s.\n", err)
			}

		default:
			log.Fatal("Failed to decompress archive: unknown tar type flag %d for entry '%s'.\n", header.Typeflag, header.Name)
		}
	}

	metadata := metadataFile{Origin: url}
	data, err := yaml.Marshal(metadata)
	if err != nil {
		log.Fatal("Failed to marshal metadata: %s.\n", tarMetadataFileName, err)
	}
	err = ioutil.WriteFile(path.Join(modulePath, tarMetadataFileName), data, util.FileMode)
	if err != nil {
		log.Fatal("Failed to write '%s' file: %s.\n", tarMetadataFileName, err)
	}

	return TarModule{modulePath}
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
func (m TarModule) IsDirty() bool {
	return false
}

// HasOrigin returns whether the origin of the module matches `url` by
// checking the ".metadata" file inside the module directory.
func (m TarModule) HasOrigin(url string) bool {
	data, err := ioutil.ReadFile(path.Join(m.path, tarMetadataFileName))
	if err != nil {
		log.Fatal("Failed to read '%s' file: %s.\n", tarMetadataFileName, err)
	}

	var metadata metadataFile
	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		log.Fatal("Failed to unmarshal metadata: %s.\n", err)
	}
	log.Debug("Module origin is '%s'.\n", metadata.Origin)
	return metadata.Origin == url
}

// HasVersionCheckedOut returns whether the module's current version matched `version`.
// However, TarModules only have a single "master" version.
func (m TarModule) HasVersionCheckedOut(version string) bool {
	return version == tarDefaultVersion
}

// CheckoutVersion changes the module's current version to `version` if possible.
// However, TarModules only have a single "master" version. Attempting to check out any
// other version results in an error.
func (m TarModule) CheckoutVersion(version string) {
	if version == tarDefaultVersion {
		return
	}
	log.Fatal("Failed to checkout version '%s': cannot change version of TarModule.\n", version)
}

// Fetch does nothing on TarModules and reports that no changes have been fetched.
func (m TarModule) Fetch() bool {
	return false
}

// CheckedOutVersions returns all currently checked ot versions.
// For TarModules this is always the default version.
func (m TarModule) CheckedOutVersions() []string {
	return []string{tarDefaultVersion}
}
