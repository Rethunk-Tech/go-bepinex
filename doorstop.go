package bepinex

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Rethunk-Tech/go-safefs"
)

const (
	doorstopConfig = "doorstop_config.ini"
	// maxConfigBytes bounds doorstop_config.ini, which the pack's archive supplies.
	maxConfigBytes = 1 << 20
)

// runtimePathKeys are the Doorstop settings (3 and 4) that name a pack's runtime folder or file relative to the game
// folder.
var runtimePathKeys = []string{"runtimeLib", "configDir", "corlibDir", "dll_search_path_override", "dllSearchPathOverride"}

// Proxy is the Doorstop proxy library the pack in profileRoot ships; fallback before it holds a pack.
func Proxy(profileRoot, fallback string) string {
	for _, name := range proxies {
		if safefs.IsFile(filepath.Join(profileRoot, name)) {
			return name
		}
	}
	return fallback
}

// DoorstopFiles lists, relative to profileRoot and with forward slashes, the files that must sit next to the game
// executable: doorstop_config.ini and the pack's Doorstop proxy (fallback names the proxy to look for when none is
// installed). Absent ones are left out. Everything else of the pack stays in the profile.
func DoorstopFiles(profileRoot, fallback string) []string {
	var files []string
	for _, name := range []string{doorstopConfig, Proxy(profileRoot, fallback)} {
		if _, err := os.Stat(filepath.Join(profileRoot, name)); err == nil {
			files = append(files, name)
		}
	}
	return files
}

// WriteAbsoluteConfig writes the profile's doorstop_config.ini to dst with its target assembly set to the profile's
// entry (added, in the spelling of the Doorstop major version, when the config names none) and the pack's runtime
// paths made absolute. Doorstop reads relative paths from the game folder, where the profile's files are not, so the
// copy in the game folder loads the profile's BepInEx even when a start cannot pass Doorstop's flags. A path in the
// config that leaves the profile is left as written. The file's line endings are kept; proton gives paths in the Wine
// Z: form.
func WriteAbsoluteConfig(profileRoot, dst string, doorstopMajor int, proton bool) error {
	f, err := safefs.Open(filepath.Join(profileRoot, doorstopConfig))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	b, err := safefs.ReadCapped(f, maxConfigBytes)
	if err != nil {
		return err
	}
	target := HostPath(profileRoot, Entry(profileRoot), proton)
	values := map[string]string{"targetAssembly": target, "target_assembly": target}
	if HasCLR(profileRoot) {
		values["corlib_dir"] = HostPath(profileRoot, corlibDir, proton)
		values["coreclr_path"] = HostPath(profileRoot, coreclr, proton)
	}
	resolve := func(value string) (string, bool) { return profilePath(profileRoot, value, proton) }
	return safefs.WriteFileAtomic(dst, []byte(rewriteConfig(string(b), values, resolve, doorstopMajor, target)), 0o600)
}

// rewriteConfig sets each key in values and each runtimePathKeys key resolve accepts, leaving every other line as it
// was, and appends the target when no line carries it.
func rewriteConfig(text string, values map[string]string, resolve func(string) (string, bool), doorstopMajor int, target string) string {
	lines := strings.Split(text, "\n")
	found := false
	for i, line := range lines {
		key, val, ok := strings.Cut(line, "=")
		k := strings.TrimSpace(key)
		v, known := values[k]
		if ok && !known && slices.Contains(runtimePathKeys, k) {
			v, known = resolve(strings.TrimSpace(val))
		}
		if !ok || !known {
			continue
		}
		found = found || v == target
		lines[i] = k + "=" + v
		if strings.HasSuffix(line, "\r") {
			lines[i] += "\r"
		}
	}
	if !found {
		key := "targetAssembly"
		if doorstopMajor >= 4 {
			key = "target_assembly"
		}
		eol := ""
		if strings.Contains(text, "\r\n") {
			eol = "\r"
		}
		add := key + "=" + target + eol
		// Doorstop 4 reads the key from [UnityDoorstop]; at the end of the file it can land in another section.
		if i := slices.IndexFunc(lines, func(l string) bool { return strings.EqualFold(strings.TrimSpace(l), "[UnityDoorstop]") }); i >= 0 {
			lines = slices.Insert(lines, i+1, add)
		} else if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = slices.Insert(lines, n-1, add)
		} else {
			lines = append(lines, add)
		}
	}
	return strings.Join(lines, "\n")
}

// profilePath is the absolute form of a config value when it names something that exists inside the profile.
func profilePath(profileRoot, value string, proton bool) (string, bool) {
	rel := filepath.FromSlash(strings.ReplaceAll(value, `\`, "/"))
	if rel == "" || !filepath.IsLocal(rel) {
		return "", false
	}
	if _, err := os.Stat(filepath.Join(profileRoot, rel)); err != nil {
		return "", false
	}
	return HostPath(profileRoot, rel, proton), true
}
