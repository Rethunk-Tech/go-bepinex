# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's **private vulnerability reporting** on this repository (Security → Report a
vulnerability). It stays private until a fix ships.

Please do not open a public issue for a suspected vulnerability first.

## What is in scope

- **The pack zip** (`InstallPack`): no entry writes outside the profile folder, and links, special files and
  decompression bombs are refused (go-archive's checks). The extracted tree is moved by its own relative paths.
- **`manifest.json` in the zip** (`Installed.Manifest`): read only up to 1 MiB and returned unparsed.
- **`doorstop_config.ini`** (`WriteAbsoluteConfig`): read up to 1 MiB; a value naming a path outside the profile is
  never turned into an absolute path or probed for existence.
- **`user.reg`** (`wineoverride.Ensure`): read up to 128 MiB; the edit touches only the override line (or appends one
  section), keeps the line ending and every other byte, and the DLL name cannot inject another line.
- **Plugin DLLs** (`plugins.ScanDir` and the rest): untrusted bytes, parsed by go-dotnet (fuzzed there) and never loaded or
  run. `DLLs` stays inside the folder and skips symlinks.
- **`LogOutput.log` lines** (`logline.Parse`): text a mod wrote; matched with linear-time regexps and fuzzed.
- **The Unity libraries download** (`PlaceUnityLibs`): fixed host, version from digits and dots, capped at 256 MiB,
  kept only with a zip signature. The game's `globalgamemanagers` is read for its first 64 KiB only.

## What is out of scope

- Paths the caller passes (`profileRoot`, `dst`, `userReg`, `backup`, `cacheDir`) are trusted.
- The pack's own code runs inside the game by design; this module places it and does not vet it.
- Running Wine while `Ensure` edits `user.reg` loses the edit; that is documented, not a flaw.
