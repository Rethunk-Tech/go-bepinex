package bepinex

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestDetect(t *testing.T) {
	for dir, want := range map[string]Flavor{
		"be5":            FlavorBepInEx5,
		"be6mono":        FlavorBepInEx6Mono,
		"be6il2cpp":      FlavorBepInEx6IL2CPP,
		"be6il2cpp-mac":  FlavorBepInEx6IL2CPP,
		"game-mono":      FlavorNone,
		"does-not-exist": FlavorNone,
	} {
		if got := Detect(filepath.Join("testdata", dir)); got != want {
			t.Errorf("Detect(%s) = %v, want %v", dir, got, want)
		}
	}
	if s := FlavorBepInEx6IL2CPP.String(); s != "BepInEx 6 IL2CPP" {
		t.Errorf("String = %q", s)
	}
}

func TestIsIL2CPPGame(t *testing.T) {
	for dir, want := range map[string]bool{
		"game-il2cpp-win":   true,
		"game-il2cpp-linux": true,
		"game-il2cpp-mac":   true,
		"game-mono":         false,
		"game-stripped":     false,
	} {
		if got := IsIL2CPPGame(filepath.Join("testdata", dir)); got != want {
			t.Errorf("IsIL2CPPGame(%s) = %v, want %v", dir, got, want)
		}
	}
}

// An IL2CPP pack goes through the same Doorstop surface as BepInEx 5.
func TestIL2CPPPackLayout(t *testing.T) {
	p := filepath.Join("testdata", "be6il2cpp")
	if got, want := DoorstopFiles(p, "winhttp.dll"), []string{"doorstop_config.ini", "winhttp.dll"}; !slices.Equal(got, want) {
		t.Errorf("DoorstopFiles = %v, want %v", got, want)
	}
	if !HasCLR(p) || doorstopMajor(p) != 4 {
		t.Errorf("HasCLR = %v, doorstop = %d", HasCLR(p), doorstopMajor(p))
	}
	args := strings.Join(LaunchArgs(p, 4, false), " ")
	if !strings.Contains(args, "BepInEx.Unity.IL2CPP.dll") || !strings.Contains(args, "--doorstop-clr-corlib-dir") {
		t.Errorf("LaunchArgs = %s", args)
	}
	dst := filepath.Join(t.TempDir(), "doorstop_config.ini")
	if err := WriteAbsoluteConfig(p, dst, 4, false); err != nil {
		t.Fatal(err)
	}
	b, _ := safefs.ReadFile(dst)
	abs, _ := filepath.Abs(filepath.Join(p, Entry(p)))
	if !strings.Contains(string(b), "target_assembly="+abs+"\r\n") {
		t.Errorf("config = %q", b)
	}
}

func TestPlacementForDarwin(t *testing.T) {
	p := filepath.Join("testdata", "be6il2cpp-mac")
	got := PlacementFor(p, "darwin")
	if want := []string{"run_bepinex.sh", "libdoorstop.dylib"}; !slices.Equal(got.Files, want) || !slices.Equal(got.Executable, want) {
		t.Errorf("Files = %v, Executable = %v", got.Files, got.Executable)
	}
	abs, _ := filepath.Abs(filepath.Join(p, "run_bepinex.sh"))
	// LaunchOption escapes backslashes for the shell, which only shows when the test runs on Windows.
	if want := `"` + strings.ReplaceAll(abs, `\`, `\\`) + `" %command%`; got.LaunchOption != want {
		t.Errorf("LaunchOption = %q, want %q", got.LaunchOption, want)
	}
	if slices.Contains(got.Files, "winhttp.dll") {
		t.Error("darwin lists a proxy")
	}
	if none := PlacementFor(filepath.Join("testdata", "be6il2cpp"), "darwin"); none.LaunchOption != "" || len(none.Files) != 0 {
		t.Errorf("a pack without the script = %+v", none)
	}
}

func TestPlacementForOtherOSMatchesDoorstopFiles(t *testing.T) {
	p := filepath.Join("testdata", "be5")
	for _, goos := range []string{"windows", "linux"} {
		got := PlacementFor(p, goos)
		if !slices.Equal(got.Files, DoorstopFiles(p, "winhttp.dll")) || got.LaunchOption != "" || got.Executable != nil {
			t.Errorf("%s: %+v", goos, got)
		}
	}
}

func TestLaunchOptionEscapes(t *testing.T) {
	if got, want := LaunchOption(`/a b/"x"$y`), `"/a b/\"x\"\$y" %command%`; got != want {
		t.Errorf("LaunchOption = %q, want %q", got, want)
	}
}

func TestInstallPackClearsMacFiles(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	writeAt(t, filepath.Join(profile, "run_bepinex.sh"), []byte("old"))
	writeAt(t, filepath.Join(profile, "libdoorstop.dylib"), []byte("old"))
	if _, err := InstallPack(buildPack(t, "3.0"), profile); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"run_bepinex.sh", "libdoorstop.dylib"} {
		if _, err := os.Stat(filepath.Join(profile, name)); err == nil {
			t.Errorf("%s survived a pack install", name)
		}
	}
}
