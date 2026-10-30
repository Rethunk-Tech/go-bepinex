package bepinex

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
	"github.com/Rethunk-Tech/go-safefs"
)

const (
	// unityLibsURL is where BepInEx 6 fetches a Unity version's base libraries on an IL2CPP game's first start.
	unityLibsURL = "https://unity.bepinex.dev/libraries/"
	// unityHeaderBytes is how far into globalgamemanagers the version is looked for; it sits in the file's header.
	unityHeaderBytes = 1 << 16
	maxUnityLibs     = 256 << 20
)

// unityVersionPattern is the engine version Unity writes near the start of globalgamemanagers, as 2022.3.21f1.
var unityVersionPattern = regexp.MustCompile(`(\d{4}\.\d+\.\d+)[abfp]\d+`)

// UnityVersion is the Unity version of the game in gameDir (2022.3.21), read from its *_Data folder, or "".
func UnityVersion(gameDir string) string {
	dirs, _ := filepath.Glob(filepath.Join(gameDir, "*_Data"))
	for _, dir := range dirs {
		f, err := safefs.Open(filepath.Join(dir, "globalgamemanagers"))
		if err != nil {
			continue
		}
		head, _ := io.ReadAll(io.LimitReader(f, unityHeaderBytes))
		_ = f.Close()
		if m := unityVersionPattern.FindSubmatch(head); m != nil {
			return string(m[1])
		}
	}
	return ""
}

// PlaceUnityLibs puts the game's Unity base libraries in the profile's BepInEx/unity-libs, so an IL2CPP game's first
// start needs no network. The zip is downloaded once into cacheDir/unity-libs and copied from there into each later
// profile. A game with no readable Unity version is a no-op; on any error BepInEx fetches the libraries itself at
// launch, so a caller may treat the error as non-fatal.
func PlaceUnityLibs(ctx context.Context, gameDir, profileDir, cacheDir string) error {
	return placeUnityLibs(ctx, unityLibsURL, gameDir, profileDir, cacheDir)
}

func placeUnityLibs(ctx context.Context, baseURL, gameDir, profileDir, cacheDir string) error {
	ver := UnityVersion(gameDir)
	if ver == "" {
		return nil
	}
	kept := filepath.Join(cacheDir, "unity-libs", ver+".zip")
	if _, err := os.Stat(kept); err != nil {
		if err := downloadUnityLibs(ctx, baseURL, ver, kept); err != nil {
			return err
		}
	}
	b, err := safefs.ReadFile(kept)
	if err != nil {
		return err
	}
	return writeIn(filepath.Join(profileDir, "BepInEx", "unity-libs", ver+".zip"), b)
}

func writeIn(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return safefs.WriteFileAtomic(path, b, 0o600)
}

func downloadUnityLibs(ctx context.Context, baseURL, ver, dst string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+ver+".zip", nil)
	if err != nil {
		return err
	}
	b, err := httpkit.DoCapped(nil, req, maxUnityLibs, nil)
	if err != nil {
		return fmt.Errorf("unity libraries %s: %w", ver, err)
	}
	if !bytes.HasPrefix(b, []byte("PK")) {
		return fmt.Errorf("unity libraries %s: not a zip", ver)
	}
	return writeIn(dst, b)
}
