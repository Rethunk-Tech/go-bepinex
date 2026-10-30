//go:build unix

package plugins

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func write(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "mod.dll"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDLLsStaysInsideTheFolder(t *testing.T) {
	outside := t.TempDir()
	write(t, outside, "Elsewhere.dll", []byte("x"))
	write(t, outside, "sub/Deep.dll", []byte("x"))
	dir := t.TempDir()
	write(t, dir, "Mod.dll", []byte("x"))
	write(t, dir, "sub/Other.DLL", []byte("x"))
	write(t, dir, "readme.txt", []byte("x"))
	if err := os.Symlink(filepath.Join(outside, "Elsewhere.dll"), filepath.Join(dir, "Link.dll")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "sub"), filepath.Join(dir, "linked")); err != nil {
		t.Fatal(err)
	}
	want := []string{filepath.Join(dir, "Mod.dll"), filepath.Join(dir, "sub", "Other.DLL")}
	if got := DLLs(dir + "/./"); !slices.Equal(got, want) {
		t.Fatalf("DLLs = %v, want %v", got, want)
	}
	if got := DLLs("relative/plugins"); got != nil {
		t.Fatalf("a relative folder gave %v", got)
	}
}

func TestScanReadsDeclaredPluginsAndSkipsUnreadableDLLs(t *testing.T) {
	dll := fixture(t)
	a, b := t.TempDir(), t.TempDir()
	write(t, a, "sub/Mod.dll", dll)
	write(t, a, "Junk.dll", []byte("not an assembly"))
	write(t, b, "Empty.dll", nil)

	if each := ScanEach(a); len(each) != 2 {
		t.Fatalf("ScanEach gave %d entries for 2 DLLs", len(each))
	}
	pls := PluginsIn(a)
	if len(pls) == 0 {
		t.Fatal("no plugin read from the fixture")
	}
	if got := ScanDir(a).Plugins; !slices.Equal(got, pls) {
		t.Fatalf("ScanDir %v, PluginsIn %v", got, pls)
	}
	pl := pls[0]

	if got := PluginFile(a, pl.Name+" "+pl.Version); got != filepath.Join(a, "sub", "Mod.dll") {
		t.Errorf("PluginFile = %q", got)
	}
	if got := PluginFile(a, "nope 1.0"); got != "" {
		t.Errorf("PluginFile found %q for an unknown label", got)
	}

	owners := Owners([]string{b, a})
	for _, name := range []string{pl.GUID, pl.Name, pl.Name + " " + pl.Version} {
		if i, ok := owners[strings.ToLower(name)]; !ok || i != 1 {
			t.Errorf("owner of %q = %d, %v; want 1", name, i, ok)
		}
	}
}

func TestOwnersNamespaceNeedsOneFolder(t *testing.T) {
	dll := fixture(t)
	a, b := t.TempDir(), t.TempDir()
	write(t, a, "Mod.dll", dll)
	write(t, b, "Mod.dll", dll)
	pl := PluginsIn(a)[0]
	root, _, _ := strings.Cut(strings.ToLower(pl.Namespace), ".")
	if root == "" || root == strings.ToLower(pl.GUID) || root == strings.ToLower(pl.Name) {
		t.Skip("fixture's namespace is empty or already a named owner")
	}
	if _, ok := Owners([]string{a, b})[root]; ok {
		t.Errorf("namespace %q claimed though two folders use it", root)
	}
	if i, ok := Owners([]string{a, t.TempDir()})[root]; !ok || i != 0 {
		t.Errorf("namespace %q owner = %d, %v; want 0", root, i, ok)
	}
}
