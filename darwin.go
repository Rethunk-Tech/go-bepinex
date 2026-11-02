package bepinex

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

const macScript = "run_bepinex.sh"

// Placement is what a platform's start needs from the pack installed in a profile. It is data: the caller copies the
// files, sets the modes and applies the launch option.
type Placement struct {
	// Files are the profile files, relative with forward slashes, that the start needs. On Windows and Linux they sit
	// beside the game executable (DoorstopFiles); on darwin they stay in the profile, since the launch option names
	// the script by its absolute path.
	Files []string
	// Executable are the Files that need the execute bit (0o755); archives do not always keep it. Darwin only.
	Executable []string
	// LaunchOption is the Steam launch option that starts the game through the pack's script, or "" when the pack has
	// none or the platform has no such option. Darwin only.
	LaunchOption string
}

// PlacementFor is the Placement of the pack in profileRoot for goos ("windows", "linux", "darwin"). Windows and Linux
// give DoorstopFiles, with the usual proxy named as the fallback. darwin has no winhttp.dll proxy: Doorstop is
// injected by dyld from libdoorstop.dylib (at the pack's top or under doorstop_libs/) when run_bepinex.sh starts the
// game. A pack without the script gives an empty LaunchOption, which means it cannot start on macOS.
func PlacementFor(profileRoot, goos string) Placement {
	if goos != "darwin" {
		return Placement{Files: DoorstopFiles(profileRoot, proxies[0])}
	}
	var p Placement
	if safefs.IsFile(filepath.Join(profileRoot, macScript)) {
		p.Files = append(p.Files, macScript)
		p.Executable = append(p.Executable, macScript)
		p.LaunchOption = LaunchOption(HostPath(profileRoot, macScript, false))
	}
	for _, dir := range []string{"", "doorstop_libs"} {
		libs, _ := filepath.Glob(filepath.Join(profileRoot, dir, "libdoorstop*.dylib"))
		for _, l := range libs {
			rel, _ := filepath.Rel(profileRoot, l)
			p.Files = append(p.Files, filepath.ToSlash(rel))
			p.Executable = append(p.Executable, filepath.ToSlash(rel))
		}
	}
	return p
}

// launchQuoter escapes what a double-quoted shell word still expands, since Steam passes the option through a shell.
var launchQuoter = strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`, "`", "\\`")

// LaunchOption is the Steam launch option that runs the game through the script at the absolute path script:
// `"<script>" %command%`.
func LaunchOption(script string) string {
	return `"` + launchQuoter.Replace(script) + `" %command%`
}

var executableName = regexp.MustCompile(`(?m)^([ \t]*executable_name[ \t]*=[ \t]*).*$`)

// SetScriptExecutable sets executable_name in the pack's run_bepinex.sh to name, the game's app name (the .app folder
// without its suffix), single-quoted so the shell reads it literally. A script without the variable comes back as is.
func SetScriptExecutable(script []byte, name string) []byte {
	quoted := "'" + strings.ReplaceAll(name, "'", `'\''`) + "'"
	return executableName.ReplaceAll(script, []byte("${1}"+strings.ReplaceAll(quoted, "$", "$$")))
}
