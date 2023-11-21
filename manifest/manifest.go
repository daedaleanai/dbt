package manifest

import (
	"fmt"

	"github.com/daedaleanai/dbt/v3/log"
	"github.com/daedaleanai/dbt/v3/module"
	"github.com/daedaleanai/dbt/v3/util"
)

type Module struct {
	Name, Url, Hash, Type string
	Dirty                 bool
}

type DbtVersion struct {
	Major, Minor, Revision uint
}

type Manifest struct {
	DbtVersion DbtVersion
	Modules    []Module
}

type Commit struct {
	Id         string
	Title      string
	AuthorName string
}

type ModuleDiff struct {
	New, Old         Module
	AddedCommits     []Commit
	DiscardedCommits []Commit
	// May be null if no common ancestor is found
	FirstCommonAncestor *Commit
}

type DiffResult struct {
	Differ                       bool
	DbtVersion                   string
	ModifiedModules              []ModuleDiff
	AddedModules, RemovedModules []Module
}

func (v DbtVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Revision)
}

func (c Commit) String() string {
	return fmt.Sprintf("%s: %s - %s", c.Id[:7], c.Title, c.AuthorName)
}

func Generate(modules util.OrderedMap[string, module.Module], allowUncommittedChanges bool) (Manifest, error) {
	dbtVersion := util.VersionTriplet()
	manifest := Manifest{
		DbtVersion: DbtVersion{
			Major:    dbtVersion[0],
			Minor:    dbtVersion[1],
			Revision: dbtVersion[2],
		},
	}

	for _, mod := range modules.Values() {
		dirty := mod.IsDirty()
		if dirty {
			message := fmt.Sprintf("Module %q has uncommitted changes", mod.Name())
			if allowUncommittedChanges {
				log.Warning("%s\n", message)
			} else {
				return manifest, fmt.Errorf("%s", message)
			}
		}

		manifest.Modules = append(manifest.Modules, Module{
			Name:  mod.Name(),
			Url:   mod.URL(),
			Hash:  mod.Head(),
			Type:  mod.Type().String(),
			Dirty: dirty,
		})
	}

	return manifest, nil
}

func parseCommitFromRef(gitMod module.GitModule, ref string) (Commit, error) {
	result := Commit{Id: ref}
	var err error

	result.Title, err = gitMod.GetCommitTitle(ref)
	if err != nil {
		return result, err
	}

	result.AuthorName, err = gitMod.GetCommitAuthorName(ref)
	if err != nil {
		return result, err
	}

	return result, nil
}

func diffModule(newMod, oldMod Module) (ModuleDiff, error) {
	result := ModuleDiff{
		New:                 newMod,
		Old:                 oldMod,
		AddedCommits:        []Commit{},
		DiscardedCommits:    []Commit{},
		FirstCommonAncestor: nil,
	}

	oldModType, found := module.ParseModuleTypeString(oldMod.Type)
	if !found {
		return result, fmt.Errorf("Could not determine module type from string %q for module %q", oldMod.Type, oldMod.Name)
	}

	newModType, found := module.ParseModuleTypeString(newMod.Type)
	if !found {
		return result, fmt.Errorf("Could not determine module type from string %q for module %q", oldMod.Type, oldMod.Name)
	}

	dbtMod := module.OpenModuleByName(newMod.Name)
	if dbtMod.Type() != module.GitModuleType || oldModType != module.GitModuleType || newModType != module.GitModuleType {
		// Cannot resolve commits diffs for module types other than git
		return result, nil
	}

	gitMod := dbtMod.(module.GitModule)

	firstCommonAncestor, err := gitMod.GetMergeBase(oldMod.Hash, newMod.Hash)
	if err != nil {
		log.Warning("Unable to find common merge base between %q and %q for module %q\n", oldMod.Hash, newMod.Hash, newMod.Name)
		return result, nil
	}

	firstCommonAncestorCommit, err := parseCommitFromRef(gitMod, firstCommonAncestor)
	if err != nil {
		log.Warning("Unable to parse common merge base between %q and %q for module %q\n", oldMod.Hash, newMod.Hash, newMod.Name)
		return result, nil
	}
	result.FirstCommonAncestor = &firstCommonAncestorCommit

	parseCommitsUpToRef := func(ref string) ([]Commit, error) {
		commits := []Commit{}
		commitIdList, err := gitMod.GetCommitsBetweenRefs(firstCommonAncestor, ref)
		if err != nil {
			return commits, err
		}

		for _, id := range commitIdList {
			parsedCommit, err := parseCommitFromRef(gitMod, id)
			if err != nil {
				return commits, err
			}
			commits = append(commits, parsedCommit)
		}

		return commits, nil
	}

	temp, err := parseCommitsUpToRef(newMod.Hash)
	if err == nil {
		result.AddedCommits = temp
	}

	temp, err = parseCommitsUpToRef(oldMod.Hash)
	if err == nil {
		result.DiscardedCommits = temp
	}

	return result, nil
}

func Diff(newManifest, oldManifest Manifest) (DiffResult, error) {
	result := DiffResult{}

	if newManifest.DbtVersion != oldManifest.DbtVersion {
		result.Differ = true
		result.DbtVersion = fmt.Sprintf("DBT versions changed from %v to %v", oldManifest.DbtVersion, newManifest.DbtVersion)
	}

	findModByName := func(name string, modules []Module) (Module, bool) {
		for _, mod := range modules {
			if mod.Name == name {
				return mod, true
			}
		}
		return Module{}, false
	}

	// Iterate through the new modules and determine what modules have been added and changed.
	// A second pass through the old modules will allow us to determine which modules have been removed
	for _, mod := range newManifest.Modules {
		if matchingOldModule, found := findModByName(mod.Name, oldManifest.Modules); found {
			if mod != matchingOldModule {
				result.Differ = true
				moduleDiff, err := diffModule(mod, matchingOldModule)
				if err != nil {
					return result, err
				}
				result.ModifiedModules = append(result.ModifiedModules, moduleDiff)
			}
		} else {
			result.Differ = true
			result.AddedModules = append(result.AddedModules, mod)
		}
	}

	for _, mod := range oldManifest.Modules {
		if _, found := findModByName(mod.Name, newManifest.Modules); !found {
			result.Differ = true
			result.RemovedModules = append(result.RemovedModules, mod)
		}
	}

	return result, nil
}
