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

func TestGameDirWithGlobMetacharacters(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Game [Steam]")
	data := filepath.Join(dir, "Game_Data")
	if err := os.MkdirAll(filepath.Join(data, "il2cpp_data"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "GameAssembly.dll"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "globalgamemanagers"), []byte("2022.3.21f1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !IsIL2CPPGame(dir) {
		t.Error("IsIL2CPPGame false for a folder named with [ ]")
	}
	if got := UnityVersion(dir); got != "2022.3.21" {
		t.Errorf("UnityVersion = %q, want 2022.3.21", got)
	}
}

func TestRewriteConfigAppendsTargetInItsSection(t *testing.T) {
	in := "[UnityDoorstop]\r\nenabled=true\r\n\r\n[Il2Cpp]\r\ncoreclr_path=x\r\n"
	got := rewriteConfig(in, map[string]string{}, func(string) (string, bool) { return "", false }, 4, "T")
	want := "[UnityDoorstop]\r\ntarget_assembly=T\r\nenabled=true\r\n\r\n[Il2Cpp]\r\ncoreclr_path=x\r\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := rewriteConfig("a=b\n", nil, func(string) (string, bool) { return "", false }, 3, "T"); got != "a=b\ntargetAssembly=T\n" {
		t.Errorf("flat config: got %q", got)
	}
}

func TestInstallPackFailureKeepsPreviousPack(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	writeAt(t, filepath.Join(profile, "winhttp.dll"), []byte("old"))
	writeAt(t, filepath.Join(profile, "BepInEx", "core", "BepInEx.Preloader.dll", "blocker"), []byte("x"))
	if _, err := InstallPack(buildPack(t, "3.0"), profile); err == nil {
		t.Fatal("install over a non-empty directory succeeded")
	}
	if b, err := os.ReadFile(filepath.Join(profile, "winhttp.dll")); err != nil || string(b) != "old" {
		t.Errorf("previous proxy = %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(profile, "doorstop_config.ini")); err == nil {
		t.Error("new pack's file left behind")
	}
}
