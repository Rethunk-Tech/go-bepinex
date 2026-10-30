<h1 align="center">go-bepinex</h1>

<div align="center">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.3+-00ADD8?logo=go&logoColor=white)](go.mod)

</div>

---

Installs BepInEx 5 into a Unity game: the pack, the Doorstop files, launch arguments and the Proton DLL override. Package `bepinex`, module `github.com/Rethunk-Tech/go-bepinex`.

A pack zip is untrusted: it is extracted under go-archive's caps and a name that climbs out fails the install before
the profile changes. `user.reg` edits are byte-exact apart from the one override line.

## Quick start

```bash
go get github.com/Rethunk-Tech/go-bepinex
```

A worked example and the contracts: [HUMANS.md](HUMANS.md).

## Features

- `InstallPack` lays a BepInEx 5 or 6 pack into a folder, replacing the old pack but keeping mods and player config.
- `LaunchArgs` and `VanillaArgs` give the Doorstop 3 and 4 flags, with Wine `Z:` paths on request.
- `WriteAbsoluteConfig` makes a `doorstop_config.ini` that works from the game folder.
- `PlaceUnityLibs` keeps and places the Unity base libraries an IL2CPP game needs.
- `plugins.ScanDir`, `ScanEach`, `PluginsIn`, `DLLs` and `PluginFile` read which BepInEx plugins the DLLs in a folder declare, through go-dotnet; `Owners` maps the names a log uses to the folder that declares them.
- `plugins.LoadOrder` and `Loadable` give the order BepInEx 5's chainloader loads plugins in, dependencies first.
- `logline.Parse`, `SplitStamp` and `Format` read and write `LogOutput.log` header lines, with a logger mod's timestamp.
- `wineoverride.Ensure` sets a `native,builtin` DLL override in a Proton prefix's `user.reg`.

## Documentation

| Doc | Audience |
| --- | --- |
| [HUMANS.md](HUMANS.md) | Running and using the package |
| [AGENTS.md](AGENTS.md) | Internals, invariants, and why |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Process — commits, tests, docs |
| [SECURITY.md](SECURITY.md) | Vulnerability disclosure |
| [CHANGELOG.md](CHANGELOG.md) | Release notes |

## License

MIT — see [LICENSE](LICENSE). © 2026 Rethunk.Tech, LLC.
