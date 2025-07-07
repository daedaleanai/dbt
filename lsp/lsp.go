package lsp

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/daedaleanai/dbt/v3/log"

	"golang.org/x/tools/go/packages"
)

// Package represents a Go package within a dbt workspace
type Package struct {
	// Name is the package name as it appears in the package source code.
	Name string

	// ImportPath is the package path as used by the go/types package.
	ImportPath string

	// GoFiles lists the absolute file paths of the package's Go source files.
	// It may include files that should not be compiled, for example because
	// they contain non-matching build tags, are documentary pseudo-files such as
	// unsafe/unsafe.go or builtin/builtin.go, or are subject to cgo preprocessing.
	GoFiles []string

	// Import paths appearing in the package's Go source files
	Imports []string
}

type Driver struct {
	packages map[string]*Package
}

func NewDriver(packages map[string]*Package) Driver {
	return Driver{
		packages: packages,
	}
}

// HandleRequest processes a go/packages driver request
func (d *Driver) HandleRequest(request *packages.DriverRequest, patterns []string) (*packages.DriverResponse, error) {
	// Find matching packages based on patterns
	matchedPackages, err := d.findPackages(patterns)
	if err != nil {
		return nil, fmt.Errorf("finding packages: %w", err)
	}

	if len(matchedPackages) == 0 {
		return &packages.DriverResponse{
			NotHandled: true,
		}, nil
	}

	// Convert to packages.Package format
	var pkgs []*packages.Package
	var roots []string

	for _, dbtPkg := range matchedPackages {
		pkg := d.convertPackage(dbtPkg)
		pkgs = append(pkgs, pkg)
		roots = append(roots, pkg.ID)
	}

	// Resolve imports between packages
	pkgs = d.resolveImports(pkgs)

	response := &packages.DriverResponse{
		NotHandled: false,
		Roots:      roots,
		Packages:   pkgs,
	}
	return response, nil
}

func (d *Driver) appendPackageImports(relevantPkgs []*Package, importPath string) []*Package {
	unhandled := []string{importPath}
	for len(unhandled) > 0 {
		curImportPath := unhandled[0]
		unhandled = unhandled[1:]

		for _, imp := range d.packages[curImportPath].Imports {
			curPkg := d.packages[imp]
			if curPkg == nil {
				continue
			}

			if !slices.Contains(relevantPkgs, curPkg) {
				relevantPkgs = append(relevantPkgs, curPkg)
				unhandled = append(unhandled, curPkg.ImportPath)
			}
		}
	}

	return relevantPkgs
}

// findPackages finds packages matching the given patterns
func (d *Driver) findPackages(patterns []string) ([]*Package, error) {
	var matched []*Package

	for _, pattern := range patterns {
		if pattern == "./..." {
			// Return all packages in workspace
			for _, pkg := range d.packages {
				matched = append(matched, pkg)
			}
		} else if pattern == "." {
			// Return package in current directory
			wd, err := os.Getwd()
			if err != nil {
				return nil, err
			}

			for _, pkg := range d.packages {
				idx := slices.IndexFunc(pkg.GoFiles, func(file string) bool {
					return filepath.Dir(file) == wd
				})

				if idx != -1 {
					matched = append(matched, pkg)
					matched = d.appendPackageImports(matched, pkg.ImportPath)
					break
				}
			}
		} else if strings.HasSuffix(pattern, "/...") {
			// Return packages under prefix
			prefix := strings.TrimSuffix(pattern, "/...")
			for _, pkg := range d.packages {
				if pkg.ImportPath == prefix || strings.HasPrefix(pkg.ImportPath, prefix+"/") {
					matched = append(matched, pkg)
					matched = d.appendPackageImports(matched, pkg.ImportPath)
				}
			}
		} else {
			// Exact match
			if pkg, ok := d.packages[pattern]; ok {
				matched = append(matched, pkg)
				matched = d.appendPackageImports(matched, pkg.ImportPath)
			}
		}
	}

	return matched, nil
}

// convertPackage converts a dbt Package to packages.Package
func (d *Driver) convertPackage(dbtPkg *Package) *packages.Package {
	pkg := &packages.Package{
		ID:              "dbt@" + dbtPkg.ImportPath,
		Name:            dbtPkg.Name,
		PkgPath:         dbtPkg.ImportPath,
		GoFiles:         make([]string, len(dbtPkg.GoFiles)),
		CompiledGoFiles: make([]string, len(dbtPkg.GoFiles)),
		Imports:         make(map[string]*packages.Package),
	}

	for i, file := range dbtPkg.GoFiles {
		pkg.GoFiles[i] = file
		pkg.CompiledGoFiles[i] = file
	}
	return pkg
}

// resolveImports creates stub packages for imports and links them
func (d *Driver) resolveImports(pkgs []*packages.Package) []*packages.Package {
	// Create a map of all packages by import path
	packageMap := make(map[string]*packages.Package)
	for _, pkg := range pkgs {
		packageMap[pkg.PkgPath] = pkg
	}

	var unknownImports []string

	// Resolve imports for each package
	for _, pkg := range pkgs {
		// Find the corresponding dbt package
		dbtPkg, ok := d.packages[pkg.PkgPath]
		if !ok {
			continue
		}

		for _, importPath := range dbtPkg.Imports {
			if importPkg, exists := packageMap[importPath]; exists {
				// Link to existing package
				pkg.Imports[importPath] = importPkg
			} else {
				if !slices.Contains(unknownImports, importPath) {
					unknownImports = append(unknownImports, importPath)
				}
				// Create stub package for external import
				pkg.Imports[importPath] = nil
			}
		}
	}

	extraPkgs := LoadExtraPatterns(unknownImports)

	for _, pkg := range pkgs {
		for importPath, importedPkg := range pkg.Imports {
			if importedPkg != nil {
				continue
			}

			idx := slices.IndexFunc(extraPkgs, func(entry *packages.Package) bool {
				return entry.PkgPath == importPath
			})
			if idx == -1 {
				// Pkg was also not found in the go stdlib... Let's just add a stub...
				pkg.Imports[importPath] = &packages.Package{
					ID:      importPath,
					PkgPath: importPath,
				}
			}

			pkg.Imports[importPath] = extraPkgs[idx]
		}
	}

	pkgs = append(pkgs, extraPkgs...)

	return pkgs
}

func LoadExtraPatterns(pats []string) []*packages.Package {
	env := os.Environ()
	if idx := slices.IndexFunc(env, func(s string) bool { return strings.HasPrefix(s, "GOPACKAGESDRIVER=") }); idx != -1 {
		env[idx] = "GOPACKAGESDRIVER=off"
	} else {
		env = append(env, "GOPACKAGESDRIVER=off")
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.LoadImports,
		Tests: false,
		// We wanna use go list for this
		Env: env,
	}, pats...)
	if err != nil {
		log.Fatal("Error loading packages: %s", err)
	}
	return pkgs
}
