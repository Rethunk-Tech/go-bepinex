package plugins

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	dotnet "github.com/Rethunk-Tech/go-dotnet"
	"github.com/Rethunk-Tech/go-safefs"
)

// PluginsIn lists the plugins every DLL under dir declares.
func PluginsIn(dir string) []dotnet.Plugin { return ScanDir(dir).Plugins }

// ScanDir reads what every DLL under dir declares about its plugins.
func ScanDir(dir string) dotnet.Declared {
	var out dotnet.Declared
	for _, d := range ScanEach(dir) {
		out.Plugins = append(out.Plugins, d.Plugins...)
		out.Relations = append(out.Relations, d.Relations...)
	}
	return out
}

// ScanEach reads what each DLL under dir declares, one Declared per DLL. A DLL go-dotnet cannot read declares nothing.
func ScanEach(dir string) []dotnet.Declared {
	var out []dotnet.Declared
	eachDLL(dir, func(_ string, d dotnet.Declared) bool {
		out = append(out, d)
		return true
	})
	return out
}

// DLLs lists every DLL under dir, which must be absolute. The walk stays inside dir and keeps only regular files, so a
// symlinked DLL or folder that points elsewhere is never read.
func DLLs(dir string) []string {
	var out []string
	withRoot(dir, func(root *os.Root) {
		for _, rel := range dllsIn(root) {
			out = append(out, filepath.Join(filepath.Clean(dir), filepath.FromSlash(rel)))
		}
	})
	return out
}

// withRoot opens dir (absolute) as a root for fn, or does nothing.
func withRoot(dir string, fn func(*os.Root)) {
	dir = filepath.Clean(dir)
	if !filepath.IsAbs(dir) {
		return
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return
	}
	defer func() { _ = root.Close() }()
	fn(root)
}

// dllsIn lists the regular DLL files under root, slash-separated and relative to it.
func dllsIn(root *os.Root) []string {
	var out []string
	_ = fs.WalkDir(root.FS(), ".", func(rel string, d fs.DirEntry, err error) error {
		if err != nil {
			return fs.SkipDir
		}
		if d.Type().IsRegular() && strings.EqualFold(filepath.Ext(rel), ".dll") {
			out = append(out, rel)
		}
		return nil
	})
	return out
}

// eachDLL scans every DLL under dir and calls fn with its absolute path and what it declares, until fn returns false.
// The plugins folder is opened once and each DLL is read through that root, so a file or folder swapped for a link
// after the walk is read only if the link stays inside dir. go-dotnet scans by path, so the bytes read through the root
// are scanned from a private copy.
func eachDLL(dir string, fn func(path string, d dotnet.Declared) bool) {
	withRoot(dir, func(root *os.Root) {
		tmp, err := os.MkdirTemp("", "bepinex-scan-")
		if err != nil {
			return
		}
		defer func() { _ = os.RemoveAll(tmp) }()
		copyPath := filepath.Join(tmp, "scan.dll")
		for _, rel := range dllsIn(root) {
			var d dotnet.Declared
			if copyOut(root, rel, copyPath) == nil {
				d, _ = dotnet.Scan(copyPath)
			}
			if !fn(filepath.Join(filepath.Clean(dir), filepath.FromSlash(rel)), d) {
				return
			}
		}
	})
}

// copyOut copies the regular file rel of root to dst, at most go-dotnet's assembly cap.
func copyOut(root *os.Root, rel, dst string) error {
	src, err := root.Open(rel)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	if info, err := src.Stat(); err != nil || !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}
	b, err := io.ReadAll(io.LimitReader(src, dotnet.MaxAssemblyBytes+1))
	if err != nil {
		return err
	}
	if int64(len(b)) > dotnet.MaxAssemblyBytes {
		return dotnet.ErrTooLarge
	}
	return safefs.WriteFile(dst, b, 0o600)
}

// Owners maps every name a BepInEx log gives a plugin, lower-cased, to the index in dirs of the folder whose DLLs
// declare it: its GUID, its name and "name version". An exception's stack names a plugin only by its code's root
// namespace, so that is a name too when exactly one folder's plugins use it.
func Owners(dirs []string) map[string]int {
	owners := map[string]int{}
	byNamespace := map[string][]int{}
	for i, dir := range dirs {
		for _, pl := range PluginsIn(dir) {
			for _, name := range []string{pl.GUID, pl.Name, pl.Name + " " + pl.Version} {
				owners[strings.ToLower(name)] = i
			}
			if root, _, _ := strings.Cut(pl.Namespace, "."); root != "" {
				ns := strings.ToLower(root)
				if !slices.Contains(byNamespace[ns], i) {
					byNamespace[ns] = append(byNamespace[ns], i)
				}
			}
		}
	}
	for ns, is := range byNamespace {
		if _, named := owners[ns]; !named && len(is) == 1 {
			owners[ns] = is[0]
		}
	}
	return owners
}

// PluginFile is the DLL under dir that declares the plugin a BepInEx log names as "name version", or "" for none.
func PluginFile(dir, label string) string {
	var found string
	eachDLL(dir, func(path string, d dotnet.Declared) bool {
		for _, pl := range d.Plugins {
			if strings.EqualFold(pl.Name+" "+pl.Version, label) {
				found = path
				return false
			}
		}
		return true
	})
	return found
}
