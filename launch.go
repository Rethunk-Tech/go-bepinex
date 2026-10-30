package bepinex

import (
	"os"
	"path/filepath"

	"github.com/Rethunk-Tech/go-bepinex/wineoverride"
)

// entries are the assemblies Doorstop can start BepInEx through, relative to the profile root, which BepInEx takes its
// root from: BepInEx 5's preloader, BepInEx 6's IL2CPP entry (which runs on the .NET runtime its pack ships in dotnet/),
// BepInEx 6's Mono preloader, and the IL2CPP entry of the 6.0.0-be.577 build, which runs on the Mono runtime it ships
// in mono/.
var entries = []string{
	filepath.Join("BepInEx", "core", "BepInEx.Preloader.dll"),
	filepath.Join("BepInEx", "core", "BepInEx.Unity.IL2CPP.dll"),
	filepath.Join("BepInEx", "core", "BepInEx.Unity.Mono.Preloader.dll"),
	filepath.Join("BepInEx", "core", "BepInEx.IL2CPP.dll"),
}

// The .NET runtime of an IL2CPP pack, relative to the profile root: Doorstop 4 reads the corlib folder and coreclr
// from its [Il2Cpp] settings or its CLR flags.
const corlibDir = "dotnet"

var coreclr = filepath.Join("dotnet", "coreclr.dll")

// Entry is the entry assembly of the pack installed in profileRoot, relative to it and in the host's separators; the
// BepInEx 5 preloader when no entry assembly is installed.
func Entry(profileRoot string) string {
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(profileRoot, e)); err == nil {
			return e
		}
	}
	return entries[0]
}

// HasCLR reports whether the pack in profileRoot ships the .NET runtime an IL2CPP pack starts through.
func HasCLR(profileRoot string) bool {
	_, err := os.Stat(filepath.Join(profileRoot, coreclr))
	return err == nil
}

// HostPath is rel under profileRoot as an absolute path, in the Wine Z: form when proton is set.
func HostPath(profileRoot, rel string, proton bool) string {
	p, err := filepath.Abs(filepath.Join(profileRoot, rel))
	if err != nil {
		p = filepath.Join(profileRoot, rel)
	}
	if proton {
		return wineoverride.ZPath(p)
	}
	return p
}

// LaunchArgs are the Doorstop flags that start the game with BepInEx rooted in profileRoot, with an IL2CPP pack's .NET
// runtime. Doorstop 3 and 4 spell the flags differently (doorstopMajor is Installed.Doorstop); proton gives the paths
// in the Wine Z: form.
func LaunchArgs(profileRoot string, doorstopMajor int, proton bool) []string {
	target := HostPath(profileRoot, Entry(profileRoot), proton)
	if doorstopMajor >= 4 {
		args := []string{"--doorstop-enabled", "true", "--doorstop-target-assembly", target}
		if HasCLR(profileRoot) {
			args = append(args,
				"--doorstop-clr-corlib-dir", HostPath(profileRoot, corlibDir, proton),
				"--doorstop-clr-runtime-coreclr-path", HostPath(profileRoot, coreclr, proton))
		}
		return args
	}
	return []string{"--doorstop-enable", "true", "--doorstop-target", target}
}

// VanillaArgs start the game with Doorstop switched off, even while its files sit in the game folder. A vanilla start
// has no profile to read the Doorstop version from, and the proxy left in the folder may be either, so both spellings
// are passed; each proxy ignores the other's.
func VanillaArgs() []string {
	return []string{"--doorstop-enable", "false", "--doorstop-enabled", "false"}
}
