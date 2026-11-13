package bepinex

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func writeZip(t *testing.T, files map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "pack.zip")
	if err := safefs.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func writeAt(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := safefs.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func buildPack(t *testing.T, doorstop string) string { return buildPackIn(t, "BepInExPack", doorstop) }

// buildPackIn builds a pack whose files sit under root, as a community build names it.
func buildPackIn(t *testing.T, root, doorstop string) string {
	t.Helper()
	return buildPackWithProxy(t, root, doorstop, "winhttp.dll")
}

func buildPackWithProxy(t *testing.T, root, doorstop, proxy string) string {
	t.Helper()
	files := map[string]string{
		"manifest.json":                      `{"name":"BepInExPack","version_number":"5.4.2305"}`,
		"README.md":                          "pack",
		proxy:                                "proxy",
		"doorstop_config.ini":                "[UnityDoorstop]",
		".doorstop_version":                  doorstop,
		"doorstop_libs/x64/libdoorstop.so":   "lib",
		"BepInEx/core/BepInEx.Preloader.dll": "pre",
	}
	out := map[string]string{}
	for name, body := range files {
		if name == ".doorstop_version" && doorstop == "" {
			continue
		}
		if root != "" && name != "manifest.json" && name != "README.md" {
			name = root + "/" + name
		}
		out[name] = body
	}
	return writeZip(t, out)
}

func TestInstallPackAndDoorstopFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	// A reinstall replaces in place; "" is a pack with BepInEx/ at the zip's top.
	for _, packDir := range []string{"BepInExPack", "BepInExPack", "BepInExPack_Valheim", ""} {
		got, err := InstallPack(buildPackIn(t, packDir, "4.3.0.0\n"), root)
		if err != nil || got.Doorstop != 4 || !strings.Contains(string(got.Manifest), "5.4.2305") {
			t.Fatalf("%s: got %+v, %v", packDir, got, err)
		}
	}
	if !safefs.IsDir(filepath.Join(root, "BepInEx", "core")) {
		t.Fatal("BepInEx/core missing")
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.json")); err == nil {
		t.Fatal("pack metadata leaked into the profile")
	}
	if got := strings.Join(DoorstopFiles(root, "winhttp.dll"), ","); got != "doorstop_config.ini,winhttp.dll" {
		t.Fatalf("doorstop files %v", got)
	}
	if got, _ := InstallPack(buildPack(t, "garbage"), filepath.Join(t.TempDir(), "p")); got.Doorstop != 3 {
		t.Fatalf("default doorstop %d", got.Doorstop)
	}
}

func TestInstallPackThroughASymlinkOntoAnotherDevice(t *testing.T) {
	targetRoot, err := os.MkdirTemp("/dev/shm", "bepinex-profile-*")
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(targetRoot) })
	root := filepath.Join(t.TempDir(), "profile")
	if err := os.Symlink(targetRoot, root); err != nil {
		t.Fatal(err)
	}
	writeAt(t, filepath.Join(root, "winhttp.dll"), []byte("old-proxy"))
	if _, err := InstallPack(buildPack(t, "4.3.0.0\n"), root); err != nil {
		t.Fatal(err)
	}
	if !safefs.IsDir(filepath.Join(root, "BepInEx", "core")) {
		t.Fatal("pack not installed")
	}
}

func TestInstallPackRefusesWhatClimbsOut(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "profile")
	zipPath := writeZip(t, map[string]string{
		"BepInExPack/BepInEx/core/BepInEx.Preloader.dll": "pre",
		"BepInExPack/../../evil.dll":                     "evil",
	})
	if _, err := InstallPack(zipPath, root); err == nil {
		t.Fatal("a traversing name installed")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(parent), "evil.dll")); err == nil {
		t.Fatal("a file escaped")
	}
	ents, _ := os.ReadDir(parent)
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".bepinex-pack-") {
			t.Fatalf("scratch folder %s left behind", e.Name())
		}
	}
	if _, err := InstallPack(writeZip(t, map[string]string{"Other/x.dll": "x"}), root); err == nil {
		t.Fatal("a zip with no entry assembly installed")
	}
}

func TestAPackKeepsANestedConfigFile(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	writeAt(t, filepath.Join(root, "BepInEx", "config", "Mod", "settings.cfg"), []byte("mine"))
	zipPath := writeZip(t, map[string]string{
		"manifest.json":                                  `{"name":"BepInExPack","version_number":"5.4.2305"}`,
		"BepInExPack/winhttp.dll":                        "proxy",
		"BepInExPack/BepInEx/core/BepInEx.Preloader.dll": "x",
		"BepInExPack/BepInEx/config/Mod/settings.cfg":    "pack's",
	})
	if _, err := InstallPack(zipPath, root); err != nil {
		t.Fatal(err)
	}
	if b, _ := safefs.ReadFile(filepath.Join(root, "BepInEx", "config", "Mod", "settings.cfg")); string(b) != "mine" {
		t.Fatalf("nested config replaced: %q", b)
	}
}

func TestAnOlderPackOverANewerOneKeepsNoneOfItsFiles(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile")
	if _, err := InstallPack(buildPack(t, "4.3.0.0\n"), root); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"BepInEx/core/Stale.dll":         "newer pack only",
		"BepInEx/core/Ns-Mod/Mod.dll":    "a package's core file",
		"BepInEx/config/BepInEx.cfg":     "[Logging.Console]\nEnabled = true\n",
		"BepInEx/plugins/Ns-Mod/Mod.dll": "plugin",
	} {
		writeAt(t, filepath.Join(root, name), []byte(body))
	}
	got, err := InstallPack(buildPack(t, ""), root)
	if err != nil || got.Doorstop != 3 {
		t.Fatalf("an older pack without .doorstop_version is Doorstop 3: %+v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(root, "BepInEx", "core", "Stale.dll")); err == nil {
		t.Fatal("the newer pack's core file was left behind")
	}
	for _, kept := range []string{"BepInEx/core/Ns-Mod/Mod.dll", "BepInEx/plugins/Ns-Mod/Mod.dll", "BepInEx/core/BepInEx.Preloader.dll"} {
		if _, err := os.Stat(filepath.Join(root, kept)); err != nil {
			t.Fatalf("%s: %v", kept, err)
		}
	}
	if b, _ := safefs.ReadFile(filepath.Join(root, "BepInEx", "config", "BepInEx.cfg")); !strings.Contains(string(b), "Enabled = true") {
		t.Fatalf("the player's BepInEx.cfg was replaced: %q", b)
	}
}

// An IL2CPP pack starts through its own entry assembly on the .NET runtime it ships, which Doorstop finds only through
// absolute paths, since it reads relative ones from the game folder.
func TestAnIL2CPPPackStartsThroughItsRuntime(t *testing.T) {
	zipPath := writeZip(t, map[string]string{
		"manifest.json":                                     `{"name":"BepInExPack_IL2CPP","version_number":"6.0.788"}`,
		"BepInExPack/winhttp.dll":                           "proxy",
		"BepInExPack/.doorstop_version":                     "4.5.0",
		"BepInExPack/doorstop_config.ini":                   "[General]\r\ntarget_assembly = BepInEx\\core\\BepInEx.Unity.IL2CPP.dll\r\n[Il2Cpp]\r\ncoreclr_path = dotnet\\coreclr.dll\r\ncorlib_dir = dotnet\r\n",
		"BepInExPack/BepInEx/core/BepInEx.Unity.IL2CPP.dll": "entry",
		"BepInExPack/dotnet/coreclr.dll":                    "clr",
	})
	profile := filepath.Join(t.TempDir(), "profile")
	got, err := InstallPack(zipPath, profile)
	if err != nil || got.Doorstop != 4 {
		t.Fatalf("install %+v %v", got, err)
	}
	if Entry(profile) != filepath.Join("BepInEx", "core", "BepInEx.Unity.IL2CPP.dll") || !HasCLR(profile) {
		t.Fatal("an IL2CPP pack is not recognised")
	}
	dst := filepath.Join(t.TempDir(), "abs.ini")
	if err := WriteAbsoluteConfig(profile, dst, got.Doorstop, false); err != nil {
		t.Fatal(err)
	}
	cfg, _ := safefs.ReadFile(dst)
	abs := filepath.Join(profile, "dotnet")
	for _, line := range []string{"corlib_dir=" + abs + "\r", "coreclr_path=" + filepath.Join(abs, "coreclr.dll") + "\r"} {
		if !strings.Contains(string(cfg), line) {
			t.Fatalf("config lacks %q:\n%s", line, cfg)
		}
	}
	if args := strings.Join(LaunchArgs(profile, 4, false), " "); !strings.Contains(args, "--doorstop-clr-corlib-dir "+abs) {
		t.Fatalf("args %q", args)
	}
	if _, err := InstallPack(buildPack(t, "4.3.0.0"), profile); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(profile, "dotnet")); err == nil || HasCLR(profile) {
		t.Fatal("a BepInEx 5 pack over an IL2CPP one kept its runtime")
	}
}

// BepInEx 6 Mono (a zip written with backslash separators) and the legacy IL2CPP build on a shipped Mono runtime start
// through their own entry assembly, and their config's relative runtime folders point into the profile.
func TestPacksWithTheirOwnRuntimeFolders(t *testing.T) {
	cases := []struct {
		name, sep, entry, config string
		files                    []string
		want                     []string
	}{
		{
			"mono", `\`, `BepInEx\core\BepInEx.Unity.Mono.Preloader.dll`,
			"[General]\ntarget_assembly = BepInEx\\core\\BepInEx.Unity.Mono.Preloader.dll\n[UnityMono]\ndll_search_path_override = unstripped_corlib\n",
			[]string{`unstripped_corlib\mscorlib.dll`},
			[]string{"dll_search_path_override=%s/unstripped_corlib", "target_assembly=%s/BepInEx/core/BepInEx.Unity.Mono.Preloader.dll"},
		},
		{
			"il2cpp-on-mono", "/", "BepInEx/core/BepInEx.IL2CPP.dll",
			"[UnityDoorstop]\ntargetAssembly=BepInEx\\core\\BepInEx.IL2CPP.dll\n[MonoBackend]\nruntimeLib=mono\\EmbedRuntime\\mono-2.0-sgen.dll\nconfigDir=mono\\etc\ncorlibDir=mono\\Managed\n",
			[]string{"mono/EmbedRuntime/mono-2.0-sgen.dll", "mono/etc/machine.config", "mono/Managed/mscorlib.dll"},
			[]string{"runtimeLib=%s/mono/EmbedRuntime/mono-2.0-sgen.dll", "configDir=%s/mono/etc", "corlibDir=%s/mono/Managed", "targetAssembly=%s/BepInEx/core/BepInEx.IL2CPP.dll"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			add := func(name, body string) {
				w, _ := zw.Create("BepInExPack" + c.sep + name)
				_, _ = w.Write([]byte(body))
			}
			if c.sep != "/" {
				add(`BepInEx\`, "")
			}
			add("winhttp.dll", "proxy")
			add("doorstop_config.ini", c.config)
			add(c.entry, "entry")
			for _, f := range c.files {
				add(f, "x")
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			zipPath := filepath.Join(t.TempDir(), "pack.zip")
			if err := safefs.WriteFile(zipPath, buf.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			profile := filepath.Join(t.TempDir(), "profile")
			if _, err := InstallPack(zipPath, profile); err != nil {
				t.Fatal(err)
			}
			dst := filepath.Join(profile, "abs.ini")
			if err := WriteAbsoluteConfig(profile, dst, 4, false); err != nil {
				t.Fatal(err)
			}
			cfg, _ := safefs.ReadFile(dst)
			abs := filepath.ToSlash(profile)
			for _, line := range c.want {
				if !strings.Contains(strings.ReplaceAll(string(cfg), `\`, "/"), fmt.Sprintf(line, abs)) {
					t.Errorf("config lacks %q:\n%s", fmt.Sprintf(line, abs), cfg)
				}
			}
		})
	}
}

func TestAConfigPathThatLeavesTheProfileIsLeftAsWritten(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	writeAt(t, filepath.Join(profile, "doorstop_config.ini"), []byte("[General]\ncorlibDir=..\\outside\n"))
	writeAt(t, filepath.Join(filepath.Dir(profile), "outside", "x"), []byte("x"))
	dst := filepath.Join(profile, "abs.ini")
	if err := WriteAbsoluteConfig(profile, dst, 3, false); err != nil {
		t.Fatal(err)
	}
	if cfg, _ := safefs.ReadFile(dst); !strings.Contains(string(cfg), "corlibDir=..\\outside\n") {
		t.Fatalf("config:\n%s", cfg)
	}
}

func TestLaunchArgs(t *testing.T) {
	root := filepath.Join(t.TempDir(), "profile one")
	pre := filepath.Join(root, "BepInEx", "core", "BepInEx.Preloader.dll")
	cases := []struct {
		major  int
		proton bool
		want   string
	}{
		{3, false, "--doorstop-enable true --doorstop-target " + pre},
		{4, false, "--doorstop-enabled true --doorstop-target-assembly " + pre},
	}
	if runtime.GOOS != "windows" { // the Wine Z: drive maps the host's root, which only a Unix host has
		cases = append(cases, struct {
			major  int
			proton bool
			want   string
		}{4, true, "--doorstop-enabled true --doorstop-target-assembly Z:" + strings.ReplaceAll(pre, "/", `\`)})
	}
	for _, c := range cases {
		if got := strings.Join(LaunchArgs(root, c.major, c.proton), " "); got != c.want {
			t.Errorf("LaunchArgs(%d,%v) = %q, want %q", c.major, c.proton, got, c.want)
		}
	}
	if strings.Join(VanillaArgs(), " ") != "--doorstop-enable false --doorstop-enabled false" {
		t.Fatal("vanilla args")
	}
}

func TestAPackWhoseProxyIsVersionDllLoadsThatOne(t *testing.T) {
	profile := filepath.Join(t.TempDir(), "profile")
	if _, err := InstallPack(buildPackWithProxy(t, "BepInExPack", "4.5.0", "version.dll"), profile); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(DoorstopFiles(profile, "winhttp.dll"), " "); got != "doorstop_config.ini version.dll" {
		t.Fatalf("placed %q", got)
	}
	if Proxy(profile, "winhttp.dll") != "version.dll" || Proxy(t.TempDir(), "winmm.dll") != "winmm.dll" {
		t.Fatal("Proxy")
	}
	// An older pack with the usual proxy replaces the renamed one rather than leaving both.
	if _, err := InstallPack(buildPack(t, "4.3.0.0"), profile); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(DoorstopFiles(profile, "winhttp.dll"), " "); got != "doorstop_config.ini winhttp.dll" || safefs.IsFile(filepath.Join(profile, "version.dll")) {
		t.Fatalf("placed %q after the swap", got)
	}
}

func TestInstallPackRefusesWhatIsNotAPack(t *testing.T) {
	dir := t.TempDir()
	notZip := filepath.Join(dir, "x.zip")
	writeAt(t, notZip, []byte("not a zip"))
	for name, zipPath := range map[string]string{
		"missing":      filepath.Join(dir, "none.zip"),
		"not a zip":    notZip,
		"no pack root": writeZip(t, map[string]string{"readme.txt": "hi"}),
	} {
		profile := filepath.Join(t.TempDir(), "profile")
		if _, err := InstallPack(zipPath, profile); err == nil {
			t.Errorf("%s: installed", name)
		}
	}
	// The profile's parent is a file, so even its folder cannot be made.
	if _, err := InstallPack(notZip, filepath.Join(notZip, "profile")); err == nil {
		t.Error("a profile under a file was made")
	}
}
