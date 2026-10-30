package bepinex

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

// An IL2CPP game's Unity base libraries are fetched once, kept, and placed in each profile from the kept copy.
func TestPlaceUnityLibs(t *testing.T) {
	game := t.TempDir()
	data := filepath.Join(game, "Game_Data")
	if err := os.MkdirAll(data, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "globalgamemanagers"), []byte("\x00\x00junk 2022.3.21f1 more"), 0o600); err != nil {
		t.Fatal(err)
	}
	var asked atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		if r.URL.Path != "/2022.3.21.zip" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("PK\x03\x04zip"))
	}))
	defer srv.Close()
	cache := t.TempDir()
	for _, profile := range []string{t.TempDir(), t.TempDir()} {
		if err := placeUnityLibs(t.Context(), srv.URL+"/", game, profile, cache); err != nil {
			t.Fatal(err)
		}
		if b, err := safefs.ReadFile(filepath.Join(profile, "BepInEx", "unity-libs", "2022.3.21.zip")); err != nil || string(b) != "PK\x03\x04zip" {
			t.Fatalf("placed = %q, %v", b, err)
		}
	}
	if asked.Load() != 1 {
		t.Fatalf("asked %d times, want once", asked.Load())
	}
	if UnityVersion(t.TempDir()) != "" {
		t.Fatal("a folder with no Unity data has a version")
	}
}

func TestPlaceUnityLibsRefusesWhatIsNotAZip(t *testing.T) {
	game := t.TempDir()
	writeAt(t, filepath.Join(game, "G_Data", "globalgamemanagers"), []byte("2021.3.1f1"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) }))
	defer srv.Close()
	cache := t.TempDir()
	if err := placeUnityLibs(t.Context(), srv.URL+"/", game, t.TempDir(), cache); err == nil {
		t.Fatal("a non-zip was kept")
	}
	if safefs.IsFile(filepath.Join(cache, "unity-libs", "2021.3.1.zip")) {
		t.Fatal("a non-zip reached the cache")
	}
}

func TestPublicWrappers(t *testing.T) {
	// A game with no readable Unity version returns before any download.
	if err := PlaceUnityLibs(t.Context(), t.TempDir(), t.TempDir(), t.TempDir()); err != nil {
		t.Fatal(err)
	}
	p := Proxies()
	p[0] = "changed"
	if Proxies()[0] != "winhttp.dll" {
		t.Error("Proxies hands out its own slice")
	}
}
