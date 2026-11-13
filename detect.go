package bepinex

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

// Flavor is which BepInEx build an installed pack is.
type Flavor int

const (
	// FlavorNone is a folder with no BepInEx entry assembly.
	FlavorNone Flavor = iota
	// FlavorBepInEx5 is BepInEx 5, which runs on Mono.
	FlavorBepInEx5
	// FlavorBepInEx6Mono is BepInEx 6 for a Mono game.
	FlavorBepInEx6Mono
	// FlavorBepInEx6IL2CPP is BepInEx 6 for an IL2CPP game.
	FlavorBepInEx6IL2CPP
)

// String is the flavor's name, as "BepInEx 6 IL2CPP".
func (f Flavor) String() string {
	switch f {
	case FlavorNone:
		return "none"
	case FlavorBepInEx5:
		return "BepInEx 5"
	case FlavorBepInEx6Mono:
		return "BepInEx 6 Mono"
	case FlavorBepInEx6IL2CPP:
		return "BepInEx 6 IL2CPP"
	}
	return "unknown"
}

// Detect tells which BepInEx build is installed in profileRoot from its entry assembly. When a folder holds more than
// one, the IL2CPP entry wins, then the BepInEx 6 Mono one, as Entry does.
func Detect(profileRoot string) Flavor {
	switch Entry(profileRoot) {
	case entries[1], entries[3]:
		return FlavorBepInEx6IL2CPP
	case entries[2]:
		return FlavorBepInEx6Mono
	}
	if safefs.IsFile(filepath.Join(profileRoot, entries[0])) {
		return FlavorBepInEx5
	}
	return FlavorNone
}

// il2cppGameLibs are GameAssembly's names per OS, relative to the game folder; the macOS one sits inside the .app.
var il2cppGameLibs = []string{
	"GameAssembly.dll",
	"GameAssembly.so",
	filepath.Join("Contents", "Frameworks", "GameAssembly.dylib"),
}

// IsIL2CPPGame reports whether the Unity game in gameDir is an IL2CPP build: GameAssembly (.dll, .so, or the .dylib
// of a macOS .app's folder) is there and the game's il2cpp_data folder exists. gameDir is the folder holding the
// executable, or a macOS .app. A game without both is Mono (or not a Unity game).
func IsIL2CPPGame(gameDir string) bool {
	hasLib := false
	for _, lib := range il2cppGameLibs {
		if safefs.IsFile(filepath.Join(gameDir, lib)) {
			hasLib = true
			break
		}
	}
	if !hasLib {
		return false
	}
	cands := []string{filepath.Join(gameDir, "Contents", "Resources", "Data", "il2cpp_data")}
	for _, d := range dataDirs(gameDir) {
		cands = append(cands, filepath.Join(d, "il2cpp_data"))
	}
	for _, d := range cands {
		if info, err := os.Stat(d); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// dataDirs lists the game folder's *_Data directories. It reads the folder rather than globbing, since a game folder
// named "Game [Steam]" holds glob metacharacters.
func dataDirs(gameDir string) []string {
	ents, _ := os.ReadDir(gameDir)
	var out []string
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), "_Data") {
			out = append(out, filepath.Join(gameDir, e.Name()))
		}
	}
	return out
}
