package bepinex

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	archive "github.com/Rethunk-Tech/go-archive"
	"github.com/Rethunk-Tech/go-safefs"
)

const (
	doorstopFile    = ".doorstop_version"
	defaultDoorstop = 3
	// maxManifestBytes bounds the pack's manifest.json, which comes from the untrusted archive.
	maxManifestBytes = 1 << 20
)

// proxies are the names a pack gives its Doorstop proxy, the usual one first.
var proxies = []string{"winhttp.dll", "version.dll", "winmm.dll"}

// Proxies are the file names a pack gives its Doorstop proxy library, the usual one first.
func Proxies() []string { return append([]string(nil), proxies...) }

// packRuntimeDirs are the folders a pack lays out beside BepInEx for the runtime or core libraries Doorstop loads the
// entry assembly with: the .NET runtime of an IL2CPP pack, the Mono runtime of a legacy one, and the unstripped core
// libraries of a Mono one.
var packRuntimeDirs = []string{corlibDir, "mono", "unstripped_corlib"}

// packMeta are the package metadata files a zip carries at its top, kept out of the profile when the pack's BepInEx
// files share that top.
var packMeta = map[string]bool{"manifest.json": true, "README.md": true, "CHANGELOG.md": true, "icon.png": true}

// Installed is what InstallPack found in the pack.
type Installed struct {
	// Manifest is the pack's manifest.json as the archive holds it (at most 1 MiB), or nil when it has none or it is
	// larger. It is untrusted input for the caller to parse.
	Manifest []byte
	// Doorstop is the major version of the Doorstop proxy, from .doorstop_version; 3 when the pack has none.
	Doorstop int
}

// InstallPack lays the pack zip's root folder out into profileRoot, replacing files already there: a folder named
// anything that holds an entry assembly (BepInExPack/, BepInExPack_Valheim/), or the zip itself when it has BepInEx/ at
// its top. The zip is extracted by go-archive under its default caps, so a name that climbs out, a link or a special
// file fails the install before profileRoot is touched.
//
// Files of a previous pack are cleared first (see the Doorstop files and core libraries it owns); BepInEx/plugins,
// patchers and a core folder of a mod stay, and an existing BepInEx/config file is never overwritten.
func InstallPack(zipPath, profileRoot string) (Installed, error) {
	if err := os.MkdirAll(profileRoot, 0o750); err != nil {
		return Installed{}, err
	}
	// Inside the profile so a symlink to another device does not put the scratch on the link's parent filesystem.
	tmp, err := os.MkdirTemp(profileRoot, ".bepinex-pack-*")
	if err != nil {
		return Installed{}, err
	}
	defer func() { _ = safefs.RemoveAll(tmp) }()
	if err := extract(zipPath, tmp); err != nil {
		return Installed{}, err
	}
	src, err := packRoot(tmp)
	if err != nil {
		return Installed{}, err
	}
	var manifest []byte
	if info, err := safefs.Stat(filepath.Join(tmp, "manifest.json")); err == nil && info.Mode().IsRegular() && info.Size() <= maxManifestBytes {
		manifest, _ = safefs.ReadFile(filepath.Join(tmp, "manifest.json"))
	}
	stash := filepath.Join(tmp, ".previous-pack")
	stashed, err := clearPack(profileRoot, stash)
	if err != nil {
		restorePack(profileRoot, stash, stashed, nil)
		return Installed{}, err
	}
	var placed []string
	err = filepath.WalkDir(src, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(src, p)
		if src == tmp && packMeta[rel] {
			return nil
		}
		dst := filepath.Join(profileRoot, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o750)
		}
		// A config file the profile already has holds the player's settings; BepInEx adds any new keys itself.
		if strings.HasPrefix(rel, filepath.Join("BepInEx", "config")+string(filepath.Separator)) {
			if _, err := os.Stat(dst); err == nil {
				return nil
			}
		}
		if err := safefs.Rename(p, dst); err != nil {
			return err
		}
		placed = append(placed, dst)
		return nil
	})
	if err != nil {
		restorePack(profileRoot, stash, stashed, placed)
		return Installed{}, err
	}
	return Installed{Manifest: manifest, Doorstop: doorstopMajor(profileRoot)}, nil
}

func extract(zipPath, dest string) error {
	f, err := safefs.Open(zipPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	return archive.Extract(f, info.Size(), filepath.Base(zipPath), root, archive.Options{})
}

// clearPack moves the Doorstop files and BepInEx core a previous pack laid out into stash, so a pack installed over
// another keeps none of its files: a newer pack's .doorstop_version left behind would make a caller pass Doorstop 4
// flags to an older Doorstop 3 proxy, which then never starts BepInEx. A mod's own core files sit in folders below core
// and stay. It returns the profile-relative paths it moved, for restorePack.
func clearPack(profileRoot, stash string) ([]string, error) {
	var moved []string
	take := func(rel string) error {
		if _, err := os.Lstat(filepath.Join(profileRoot, rel)); errors.Is(err, fs.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		dst := filepath.Join(stash, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return err
		}
		if err := safefs.Rename(filepath.Join(profileRoot, rel), dst); err != nil {
			return err
		}
		moved = append(moved, rel)
		return nil
	}
	for _, name := range append(append([]string{doorstopFile, doorstopConfig, "doorstop_libs", macScript, "libdoorstop.dylib"}, proxies...), packRuntimeDirs...) {
		if err := take(name); err != nil {
			return moved, err
		}
	}
	core := filepath.Join("BepInEx", "core")
	ents, err := os.ReadDir(filepath.Join(profileRoot, core))
	if errors.Is(err, fs.ErrNotExist) {
		return moved, nil
	} else if err != nil {
		return moved, err
	}
	for _, e := range ents {
		if !e.IsDir() {
			if err := take(filepath.Join(core, e.Name())); err != nil {
				return moved, err
			}
		}
	}
	return moved, nil
}

// restorePack undoes a failed install: it removes the files already placed and puts the stashed ones back. Restoring
// is best effort; the original error is what the caller needs.
func restorePack(profileRoot, stash string, stashed, placed []string) {
	for _, dst := range placed {
		_ = safefs.RemoveAll(dst)
	}
	for _, rel := range stashed {
		dst := filepath.Join(profileRoot, rel)
		_ = safefs.RemoveAll(dst)
		_ = safefs.Rename(filepath.Join(stash, rel), dst)
	}
}

// packRoot is the folder of the extracted pack that holds an entry assembly: a top-level folder, or dir itself.
func packRoot(dir string) (string, error) {
	if hasEntry(dir) {
		return dir, nil
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if e.IsDir() && hasEntry(filepath.Join(dir, e.Name())) {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = filepath.ToSlash(e)
	}
	return "", errors.New("the zip has no folder holding " + strings.Join(names, " or "))
}

func hasEntry(dir string) bool {
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(dir, e)); err == nil {
			return true
		}
	}
	return false
}

func doorstopMajor(profileRoot string) int {
	b, err := safefs.ReadFile(filepath.Join(profileRoot, doorstopFile))
	if err != nil {
		return defaultDoorstop
	}
	major, _, _ := strings.Cut(strings.TrimSpace(string(b)), ".")
	if n, err := strconv.Atoi(major); err == nil && n > 0 {
		return n
	}
	return defaultDoorstop
}
