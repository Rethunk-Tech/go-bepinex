# Changelog

Notable changes to `go-bepinex`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-11-02

### Added

- `bepinex.Detect` tells an installed BepInEx 5, BepInEx 6 Mono and BepInEx 6 IL2CPP pack apart (`Flavor`), and
  `IsIL2CPPGame` tells an IL2CPP game folder from a Mono one on Windows, Linux and macOS.
- `bepinex.PlacementFor` gives the files a start needs per OS, and for macOS the `run_bepinex.sh` and `libdoorstop.dylib`
  files that need the execute bit and the Steam launch option; `LaunchOption` quotes a script path into that option.
- `InstallPack` clears a previous pack's `run_bepinex.sh` and `libdoorstop.dylib`.

## [0.1.0] - 2026-10-30

### Added

- `bepinex.InstallPack` installs a BepInEx 5 or 6 pack zip into a profile folder, replacing the previous pack and
  keeping mods and player config, with go-archive's extraction caps.
- `bepinex.LaunchArgs`, `VanillaArgs`, `DoorstopFiles`, `Proxy`, `Entry`, `HasCLR` and `HostPath` describe how Doorstop 3
  and 4 start the pack, including an IL2CPP pack's .NET runtime and Wine `Z:` paths.
- `bepinex.WriteAbsoluteConfig` writes a `doorstop_config.ini` whose paths work from the game folder.
- `bepinex.PlaceUnityLibs` and `UnityVersion` keep and place the Unity base libraries an IL2CPP game needs.
- `wineoverride.Ensure` sets a native DLL override in a Proton prefix's `user.reg` byte-exactly; `ZPath` maps host paths
  to Wine's `Z:` form.
- `plugins.ScanDir`, `ScanEach`, `PluginsIn`, `DLLs`, `PluginFile` and `Owners` read which BepInEx plugins the DLLs in a
  folder declare through go-dotnet, without following links out of the folder.
- `plugins.LoadOrder` and `Loadable` give the order BepInEx 5's chainloader loads plugins in.
- `logline.Parse`, `SplitStamp` and `Format` read and write BepInEx `LogOutput.log` header lines, fuzzed.

[Unreleased]: https://github.com/Rethunk-Tech/go-bepinex/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Rethunk-Tech/go-bepinex/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/Rethunk-Tech/go-bepinex/releases/tag/v0.1.0
