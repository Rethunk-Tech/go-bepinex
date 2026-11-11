# Security

## Supported versions

The latest tagged release. There are no long-lived release branches, so a fix ships in the next tag
rather than being backported.

## Reporting a vulnerability

Use GitHub's private vulnerability reporting on this repository: <https://github.com/Rethunk-Tech/go-bepinex/security/advisories/new>, or the **Security** tab, then **Report a vulnerability**. The report stays private between you and us until a fix ships.

If private reporting is unavailable, email [oss@rethunk.tech](mailto:oss@rethunk.tech) with **[SECURITY]** in the subject.

Please do not open a public issue, pull request or discussion for a suspected vulnerability.

A useful report has the affected repository and version (or commit), steps to reproduce, what an attacker gets and under which conditions, and whether anyone else knows.

## What counts as a vulnerability

A vulnerability is a flaw that lets someone cross a boundary they should not: run code, read or change data they have no right to, escape a sandbox or path restriction, or leak a credential. A bug is anything else that misbehaves, such as a crash on your own input, wrong output or a slow path; open an ordinary issue for those. If you are unsure which one you have, report it privately and we will say.

## Coordinated disclosure

We handle the report inside the private advisory, so you see each status change and can comment on the fix. The advisory goes public when the fix ships, or 90 days after your report, whichever comes first. If a fix is close, either of us can ask for more time in the thread. You are credited in the advisory unless you would rather not be.

## Response timelines

- We acknowledge your report within 3 business days.
- We give you an assessment (accepted, needs more information or declined, with reasons) within 10 business days.
- We aim to ship a fix within 90 days of the report.

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

## Safe harbour

We will not pursue legal action against good-faith research that avoids privacy violations, data destruction and disruption of services, stops at a proof of concept, and keeps the details private until the advisory is published. Rethunk.Tech, LLC maintains this repository.
