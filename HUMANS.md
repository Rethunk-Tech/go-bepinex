# HUMANS.md

Run and use `go-bepinex`. Internals: [AGENTS.md](AGENTS.md).

## Quick start

```bash
go get github.com/Rethunk-Tech/go-bepinex
```

```go
package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/Rethunk-Tech/go-bepinex"
	"github.com/Rethunk-Tech/go-bepinex/wineoverride"
)

func main() {
	profile := "/home/me/profiles/default"
	pack, err := bepinex.InstallPack("BepInExPack.zip", profile)
	if err != nil {
		log.Fatal(err)
	}
	// Copy these beside the game executable.
	fmt.Println(bepinex.DoorstopFiles(profile, "winhttp.dll"))
	// Pass these to the game; true gives Wine Z: paths for Proton.
	fmt.Println(bepinex.LaunchArgs(profile, pack.Doorstop, true))

	// Proton only: load the proxy natively. The prefix's Wine must not be running.
	reg := filepath.Join("/home/me/prefix", "user.reg")
	if err := wineoverride.Ensure(reg, "winhttp", reg+".bak"); err != nil {
		log.Fatal(err)
	}
}
```

## Contracts

| Entry point | Contract |
| --- | --- |
| `InstallPack(zip, profile)` | Extracts under go-archive's caps first; on any failure `profile` is unchanged apart from being created. Replaces the old pack's Doorstop files and loose `BepInEx/core` files, keeps mod folders and existing `BepInEx/config` files. Errors when no folder in the zip holds an entry assembly. `Installed.Doorstop` is 3 without a readable `.doorstop_version`. |
| `Installed.Manifest` | The zip's `manifest.json` bytes (at most 1 MiB), unparsed, or nil. |
| `DoorstopFiles(profile, fallback)` | Slash-separated names of `doorstop_config.ini` and the proxy that exist; `fallback` names the proxy before a pack is installed. |
| `Proxy(profile, fallback)` / `Proxies()` | The installed proxy (`winhttp.dll`, `version.dll`, `winmm.dll`) or `fallback`. |
| `LaunchArgs(profile, major, proton)` | Doorstop 3 (`major < 4`) or 4 flags; adds the CLR flags when the pack ships `dotnet/coreclr.dll`. |
| `Detect(profile)` | `FlavorBepInEx5`, `FlavorBepInEx6Mono`, `FlavorBepInEx6IL2CPP` or `FlavorNone`, from the entry assembly alone. |
| `IsIL2CPPGame(dir)` | True when `GameAssembly.dll`, `.so` or a `.app`'s `Contents/Frameworks/GameAssembly.dylib` and an `il2cpp_data` folder both exist. |
| `PlacementFor(profile, goos)` | `windows`/`linux`: `DoorstopFiles`. `darwin`: `run_bepinex.sh` and `libdoorstop*.dylib` (no proxy), the same as `Executable` for the caller to chmod 0o755, and the Steam `LaunchOption` (empty when the pack has no script). Nothing is written. |
| `LaunchOption(script)` | `"<script>" %command%`, with `\`, `"`, `$` and backtick escaped. |
| `VanillaArgs()` | Switches Doorstop off in either spelling. |
| `WriteAbsoluteConfig(profile, dst, major, proton)` | Writes `dst` from the profile's config; errors if the config is missing or over 1 MiB; paths leaving the profile stay as written. |
| `Entry`, `HasCLR`, `HostPath` | The entry assembly (relative), whether a .NET runtime is shipped, and a profile path as absolute or `Z:`. |
| `UnityVersion(game)` | The `2022.3.21`-style version from `*_Data/globalgamemanagers`, or "". |
| `PlaceUnityLibs(ctx, game, profile, cache)` | No-op without a version; downloads once into `cache/unity-libs`; a failure is safe to ignore, BepInEx fetches them itself. |
| `plugins.ScanDir(dir)` / `ScanEach` / `PluginsIn` | What every DLL under the absolute `dir` declares (go-dotnet types); `ScanEach` gives one `Declared` per DLL. An unreadable DLL declares nothing. |
| `plugins.DLLs(dir)` | Regular `.dll` files under an absolute `dir`; symlinks are never followed, a relative `dir` gives nil. |
| `plugins.PluginFile(dir, "name version")` | The DLL declaring that plugin, or "". |
| `plugins.Owners(dirs)` | Lower-cased GUID, name, "name version" and a root namespace used by exactly one folder, to that folder's index. |
| `plugins.LoadOrder(assemblies)` / `Loadable(p)` | BepInEx 5.4.23's order: newest copy per GUID, versions read as `System.Version` (an unreadable one never loads), GUIDs sorted caselessly, dependencies first; `Cycle` marks plugins in a dependency cycle. |
| `logline.Parse(line)` | A `[Level  :  Source] message` header, optionally behind a `[stamp] `; false for continuation lines. |
| `logline.SplitStamp(line)` / `Format(l)` | The stamp and the rest of a line; a `Line` written back as BepInEx pads it. |
| `wineoverride.Ensure(userReg, dll, backup)` | Idempotent; refuses a `dll` that is empty or has `"`, `\`, `=` or control characters (`ErrBadName`); copies the original to `backup` once when it is not empty. |
| `wineoverride.ZPath(p)` | `Z:\...` form of a host path; meaningful on a Unix host only. |

## Requirements

Go 1.27.3 or newer, plus `lefthook` and `gitleaks` on PATH if you want the git hooks.

```bash
lefthook install
```

That is per clone; git does not commit hooks. Pre-commit scans staged changes for secrets, and
pre-push runs `gate`.

## Verify

```bash
gate
```
