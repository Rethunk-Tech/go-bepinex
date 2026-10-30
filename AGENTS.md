# AGENTS.md

Package `bepinex` installs a BepInEx pack zip into a profile folder and describes how Doorstop starts it (files,
launch flags, Unity base libraries); package `wineoverride` edits a Wine prefix's `user.reg` and maps host paths to
Wine's `Z:` form. A mod manager or launcher imports them; they import no consumer package and know no game catalogue,
profile model or Thunderstore manifest schema (the pack's `manifest.json` is returned as raw bytes).

## Gate

`gate` is the command. Each override in `.gate.toml` is there because a runner
cannot resolve that check from PATH. The `windows-latest` test job is the only place
Windows-only behaviour runs.
No code is OS-specific; the Windows job runs the same tests. The Wine `Z:` launch-argument case is skipped there,
since the `Z:` drive maps a Unix root.

## Shape

- The pack zip is read only through go-archive's `Extract`, into a scratch folder beside the profile, and the profile
  is touched only after extraction and entry-assembly detection succeed. Name safety, links and size caps are
  go-archive's; reimplementing any of it here would fork the policy.
- `InstallPack` clears the previous pack's Doorstop files and non-directory `BepInEx/core` files first, but keeps
  mod folders and an existing `BepInEx/config` file: a stale `.doorstop_version` would make a caller pass Doorstop 4
  flags to a Doorstop 3 proxy, and the config holds the player's settings.
- `Entry`, `HasCLR` and `HostPath` are the single path for "what does Doorstop start": `LaunchArgs` and
  `WriteAbsoluteConfig` both use them, so the flags and the config copy never disagree.
- `rewriteConfig` is pure so it can be fuzzed: it changes only setting lines it knows, keeps CRLF, and a config value
  resolves only if it stays inside the profile (`filepath.IsLocal`).
- `wineoverride.edit` splits and rejoins on the file's own line ending, so every untouched byte survives. A trailing
  bare CR counts as CRLF, or the appended section would flip the ending on the next run. A DLL name that could
  carry a quote, backslash, `=` or control character is refused so it cannot write another registry line.
- Size bounds are policy: `user.reg` 128 MiB, `doorstop_config.ini` and the pack manifest 1 MiB, downloaded Unity
  libraries 256 MiB.
- Network is confined to `PlaceUnityLibs`, which fetches one fixed host with a version string matched by regexp
  (digits and dots) and checks the `PK` magic before keeping the file.

## Files

| File | Holds |
| --- | --- |
| `doc.go` | Package comment |
| `install.go` | `InstallPack`, `Proxies`, pack clearing and entry-assembly detection |
| `doorstop.go` | `Proxy`, `DoorstopFiles`, `WriteAbsoluteConfig` and the config rewrite |
| `launch.go` | `Entry`, `HasCLR`, `HostPath`, `LaunchArgs`, `VanillaArgs` |
| `unitylibs.go` | `UnityVersion`, `PlaceUnityLibs` |
| `wineoverride/wineoverride.go` | `Ensure` (DllOverrides edit), `ZPath` |
