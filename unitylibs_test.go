package bepinex

import (
	"archive/zip"
	"bytes"
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
		_, _ = w.Write(unityZip(t, "UnityEngine.dll"))
	}))
	defer srv.Close()
	cache := t.TempDir()
	for _, profile := range []string{t.TempDir(), t.TempDir()} {
		if err := placeUnityLibs(t.Context(), srv.URL+"/", game, profile, cache); err != nil {
			t.Fatal(err)
		}
		if b, err := safefs.ReadFile(filepath.Join(profile, "BepInEx", "unity-libs", "2022.3.21.zip")); err != nil || !bytes.Equal(b, unityZip(t, "UnityEngine.dll")) {
			t.Fatalf("placed = %d bytes, %v", len(b), err)
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

func unityZip(t *testing.T, names ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte("x"))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func unityGame(t *testing.T) string {
	game := t.TempDir()
	writeAt(t, filepath.Join(game, "G_Data", "globalgamemanagers"), []byte("2022.3.21f1"))
	return game
}

// A redirect to another host would hand BepInEx a zip from anywhere, so the download refuses it.
func TestPlaceUnityLibsRefusesACrossHostRedirect(t *testing.T) {
	var elsewhereHit atomic.Bool
	elsewhere := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		elsewhereHit.Store(true)
		_, _ = w.Write(unityZip(t, "Evil.dll"))
	}))
	defer elsewhere.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/x.zip", http.StatusFound)
	}))
	defer srv.Close()
	cache := t.TempDir()
	if err := placeUnityLibs(t.Context(), srv.URL+"/", unityGame(t), t.TempDir(), cache); err == nil {
		t.Fatal("a cross-host redirect was followed")
	}
	if elsewhereHit.Load() || safefs.IsFile(filepath.Join(cache, "unity-libs", "2022.3.21.zip")) {
		t.Fatal("the redirect target was fetched or cached")
	}
}

func TestPlaceUnityLibsRefusesAZipThatLeavesItsFolder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(unityZip(t, "../../evil.dll"))
	}))
	defer srv.Close()
	if err := placeUnityLibs(t.Context(), srv.URL+"/", unityGame(t), t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("an escaping entry was kept")
	}
}

func TestPlaceUnityLibsRefusesABackslashZipSlip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(unityZip(t, `..\..\evil.dll`))
	}))
	defer srv.Close()
	if err := placeUnityLibs(t.Context(), srv.URL+"/", unityGame(t), t.TempDir(), t.TempDir()); err == nil {
		t.Fatal("an escaping entry was kept")
	}
}

// A cached copy that no longer checks out is dropped, not copied into the profile.
func TestPlaceUnityLibsRechecksTheCachedCopy(t *testing.T) {
	cache := t.TempDir()
	kept := filepath.Join(cache, "unity-libs", "2022.3.21.zip")
	writeAt(t, kept, []byte("PKtampered"))
	profile := t.TempDir()
	if err := placeUnityLibs(t.Context(), "http://127.0.0.1:1/", unityGame(t), profile, cache); err == nil {
		t.Fatal("a tampered cache was placed")
	}
	if safefs.IsFile(kept) || safefs.IsFile(filepath.Join(profile, "BepInEx", "unity-libs", "2022.3.21.zip")) {
		t.Fatal("a tampered cache survived")
	}
}

func TestCheckUnityZipRefusesSizeOverflow(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"a.dll", "b.dll"} {
		if _, err := zw.CreateRaw(&zip.FileHeader{
			Name:               name,
			Method:             zip.Store,
			UncompressedSize64: 1 << 63,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	err := checkUnityZip(buf.Bytes())
	if err == nil || err.Error() != "unpacks too large" {
		t.Fatalf("got %v", err)
	}
}
