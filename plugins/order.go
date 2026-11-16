package plugins

import (
	"maps"
	"slices"
	"strings"

	dotnet "github.com/Rethunk-Tech/go-dotnet"
	"github.com/Rethunk-Tech/go-semverish"
)

// Loaded is a plugin in BepInEx's load order with the dependencies of the copy that loads.
type Loaded struct {
	dotnet.Plugin
	Deps []dotnet.Relation
	// Cycle is part of a dependency cycle, on which BepInEx throws and loads no plugin at all.
	Cycle bool
}

// Skipped is a plugin the chainloader would not load.
type Skipped struct {
	Loaded
	Reason string
}

// Loadable reports whether BepInEx would load the plugin: its version must parse as a System.Version
// (Chainloader.cs, "version is invalid").
func Loadable(p dotnet.Plugin) bool {
	_, ok := semverish.CompareDotNet(p.Version, p.Version)
	return ok
}

// LoadOrder is the order BepInEx 5's chainloader loads plugins in (Chainloader.Start and Utility.TopologicalSort,
// BepInEx 5.4.23). A plugin whose version System.Version cannot read (a prerelease suffix, a leading v) is never loaded.
// Of several copies of one GUID only the newest version loads. The GUIDs are walked sorted without regard to case, and each plugin's dependencies, hard and soft, go before it, depth first, in the order its attributes
// declare them; that walk is the only tie-break. assemblies holds one Declared per DLL, so the dependencies are those of
// the copy that loads.
//
// A plugin with a missing hard dependency, a hard dependency below MinVersion, or a BepInIncompatibility whose target is
// present is omitted from the order and returned in skipped with that reason. A plugin skipped this way also skips
// plugins that hard-depend on it.
//
// ponytail: GUIDs sort by lower-cased ordinal where BepInEx uses StringComparer.InvariantCultureIgnoreCase; they differ
// only on punctuation such as '-' against letters, which no GUID in a 77-plugin real run hit.
func LoadOrder(assemblies []dotnet.Declared) (order []Loaded, skipped []Skipped) {
	newest, incompat := newestPlugins(assemblies)
	keys := slices.Sorted(maps.Keys(newest))
	var sorted []string
	cycle := map[string]bool{}
	done := map[string]bool{}
	var stack []string
	var visit func(k string)
	visit = func(k string) {
		if i := slices.Index(stack, k); i >= 0 {
			for _, c := range stack[i:] {
				cycle[c] = true
			}
			return
		}
		if done[k] {
			return
		}
		stack = append(stack, k)
		for _, dep := range newest[k].Deps {
			visit(strings.ToLower(dep.GUID))
		}
		stack = stack[:len(stack)-1]
		done[k] = true
		if _, ok := newest[k]; ok {
			sorted = append(sorted, k)
		}
	}
	for _, k := range keys {
		visit(k)
	}
	kept := map[string]Loaded{}
	seen := map[string]bool{}
	for _, k := range sorted {
		l := newest[k]
		l.Cycle = cycle[k]
		if reason := skipReason(l, kept, newest, seen, incompat[k]); reason != "" {
			skipped = append(skipped, Skipped{Loaded: l, Reason: reason})
			seen[k] = true
			continue
		}
		seen[k] = true
		kept[k] = l
		order = append(order, l)
	}
	return order, skipped
}

func skipReason(l Loaded, kept, newest map[string]Loaded, seen map[string]bool, incompat []string) string {
	for _, dep := range l.Deps {
		if dep.Kind != dotnet.HardDependency {
			continue
		}
		key := strings.ToLower(dep.GUID)
		have, ok := kept[key]
		if !ok {
			if _, present := newest[key]; !present || seen[key] {
				return "missing dependency " + dep.GUID
			}
			continue
		}
		if dep.MinVersion == "" {
			continue
		}
		if c, ok := semverish.CompareDotNet(have.Version, dep.MinVersion); !ok || c < 0 {
			return dep.GUID + " below MinVersion " + dep.MinVersion
		}
	}
	for _, g := range incompat {
		if _, ok := newest[strings.ToLower(g)]; ok {
			return "incompatible with " + g
		}
	}
	return ""
}

// newestPlugins keeps, per lower-cased GUID, the loadable plugin with the newest version, with its declared relations.
func newestPlugins(assemblies []dotnet.Declared) (map[string]Loaded, map[string][]string) {
	newest := map[string]Loaded{}
	incompat := map[string][]string{}
	for _, d := range assemblies {
		for _, p := range d.Plugins {
			if !Loadable(p) {
				continue
			}
			key := strings.ToLower(p.GUID)
			if have, ok := newest[key]; ok {
				if c, ok := semverish.CompareDotNet(p.Version, have.Version); !ok || c <= 0 {
					continue
				}
			}
			l := Loaded{Plugin: p}
			var inc []string
			for _, r := range d.Relations {
				if r.Plugin != p.GUID {
					continue
				}
				if r.Kind == dotnet.Incompatible {
					inc = append(inc, r.GUID)
					continue
				}
				l.Deps = append(l.Deps, r)
			}
			newest[key] = l
			incompat[key] = inc
		}
	}
	return newest, incompat
}
