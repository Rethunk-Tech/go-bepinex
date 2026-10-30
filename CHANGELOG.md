# Changelog

Notable changes to `go-bepinex`. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/Rethunk-Tech/go-bepinex/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Rethunk-Tech/go-bepinex/releases/tag/v0.1.0
