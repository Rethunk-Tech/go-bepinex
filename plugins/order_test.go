package plugins

import (
	"slices"
	"testing"

	dotnet "github.com/Rethunk-Tech/go-dotnet"
)

func TestLoadOrderWalksGUIDsCaselessAndPutsDependenciesFirstInDeclaredOrder(t *testing.T) {
	asm := func(guid, version string, deps ...string) dotnet.Declared {
		d := dotnet.Declared{Plugins: []dotnet.Plugin{{GUID: guid, Name: guid, Version: version}}}
		for _, g := range deps {
			d.Relations = append(d.Relations, dotnet.Relation{Plugin: guid, GUID: g, Kind: dotnet.SoftDependency})
		}
		return d
	}
	order := LoadOrder([]dotnet.Declared{
		asm("b", "1.0.0", "Z", "y"),
		asm("y", "1.0.0"),
		asm("Z", "1.0.0"),
		// An older copy's dependency does not count: the newer copy is the one that loads.
		asm("a", "1.0.0", "b"),
		asm("a", "2.0.0"),
		asm("c", "1.0.0", "d"),
		asm("d", "1.0.0", "c"),
	})
	var got []string
	for _, l := range order {
		got = append(got, l.GUID)
		if l.Cycle != (l.GUID == "c" || l.GUID == "d") {
			t.Errorf("%s cycle %v", l.GUID, l.Cycle)
		}
	}
	if want := []string{"a", "Z", "y", "b", "d", "c"}; !slices.Equal(got, want) {
		t.Fatalf("order %v, want %v", got, want)
	}
}

func TestLoadOrderReadsVersionsAsSystemVersion(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		want     []string
	}{
		{"1.0 and 1.0.0 are one version, the first stays", []string{"1.0", "1.0.0"}, []string{"1.0.0"}},
		{"prerelease never loads", []string{"1.0.0-beta", "0.9"}, []string{"0.9"}},
		{"leading v never loads", []string{"v1.0"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ds []dotnet.Declared
			for _, v := range tt.versions {
				ds = append(ds, dotnet.Declared{Plugins: []dotnet.Plugin{{GUID: "p", Name: v, Version: v}}})
			}
			var got []string
			for _, l := range LoadOrder(ds) {
				got = append(got, l.Name)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("loaded %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadable(t *testing.T) {
	for v, want := range map[string]bool{"1.0": true, "1.2.3.4": true, "1.0.0-beta": false, "v1": false, "": false} {
		if got := Loadable(dotnet.Plugin{Version: v}); got != want {
			t.Errorf("Loadable(%q) = %v, want %v", v, got, want)
		}
	}
}
