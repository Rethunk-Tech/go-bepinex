package plugins

import (
	"strings"
	"testing"

	dotnet "github.com/Rethunk-Tech/go-dotnet"
)

func FuzzLoadOrder(f *testing.F) {
	f.Add("a", "2.0.0", "b", "1.0.0", "Z", "")
	f.Add("needdep", "1.0.0", "ok", "1.0.0", "dep", "")
	f.Add("a", "1.0.0", "b", "1.0.0", "b", "")
	f.Add("needver", "1.0.0", "dep", "1.0.0", "dep", "2.0.0")
	f.Add("p", "1.0.0-beta", "p", "0.9", "", "")
	f.Add("p", "v1.0", "p", "1.0", "", "")
	f.Add("p", "1.0", "p", "1.0.0", "", "")
	f.Add("c", "1.0.0", "d", "1.0.0", "d", "")
	f.Add("mid", "1.0.0", "leaf", "1.0.0", "missing", "")
	f.Fuzz(func(t *testing.T, g1, v1, g2, v2, dep, minVer string) {
		ds := []dotnet.Declared{
			{Plugins: []dotnet.Plugin{{GUID: g1, Name: g1, Version: v1}}},
			{Plugins: []dotnet.Plugin{{GUID: g2, Name: g2, Version: v2}}},
		}
		if dep != "" && g1 != "" {
			ds[0].Relations = []dotnet.Relation{{
				Plugin: g1, GUID: dep, Kind: dotnet.HardDependency, MinVersion: minVer,
			}}
		}
		order, skipped := LoadOrder(ds)
		seen := map[string]bool{}
		for _, l := range order {
			if !Loadable(l.Plugin) {
				t.Fatalf("loaded unloadable %q %q", l.GUID, l.Version)
			}
			k := strings.ToLower(l.GUID)
			if seen[k] {
				t.Fatalf("duplicate GUID %q", l.GUID)
			}
			seen[k] = true
		}
		for _, s := range skipped {
			k := strings.ToLower(s.GUID)
			if seen[k] {
				t.Fatalf("skipped and loaded %q", s.GUID)
			}
			if !Loadable(s.Plugin) {
				t.Fatalf("skipped unloadable %q %q", s.GUID, s.Version)
			}
			if s.Reason == "" {
				t.Fatal("skipped with empty reason")
			}
			seen[k] = true
		}
	})
}
