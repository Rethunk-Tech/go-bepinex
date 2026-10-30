// Package wineoverride edits a Wine or Proton prefix's user.reg so Wine loads a loader's proxy library natively, and
// writes host paths the way a game under Wine reads them.
package wineoverride

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-safefs"
)

const (
	dllOverridesHeader = `[Software\\Wine\\DllOverrides]`
	// maxRegBytes bounds user.reg; a real one is a few MiB.
	maxRegBytes = 128 << 20
	// filetimeEpoch is the seconds between 1601 and 1970, for Wine's #time= lines.
	filetimeEpoch = 11644473600
)

// ErrBadName is returned for a library name that is not a bare DLL name.
var ErrBadName = errors.New("wineoverride: not a DLL name")

// Ensure makes the prefix's userReg load the library dll (a bare name such as winhttp, which Doorstop injects through,
// or version) natively. It is idempotent, keeps the file's line endings and every other byte, and, when backup is not
// empty, copies the file there once before the first change. dll may not contain a quote, backslash, equals sign or
// control character, which would alter another registry line. Run it while Wine is not running, or wineserver writes
// its own copy back over it.
func Ensure(userReg, dll, backup string) error {
	if !validName(dll) {
		return ErrBadName
	}
	f, err := safefs.Open(userReg)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	raw, err := safefs.ReadCapped(f, maxRegBytes)
	if err != nil {
		return err
	}
	updated, changed := edit(string(raw), dll, func() int64 { return time.Now().Unix() })
	if !changed {
		return nil
	}
	if backup != "" {
		if _, err := os.Stat(backup); errors.Is(err, fs.ErrNotExist) {
			if err := safefs.WriteFile(backup, raw, 0o600); err != nil {
				return err
			}
		}
	}
	return safefs.WriteFileAtomic(userReg, []byte(updated), 0o600)
}

func validName(dll string) bool {
	return dll != "" && !strings.ContainsFunc(dll, func(r rune) bool { return r < ' ' || r == 0x7f || r == '"' || r == '\\' || r == '=' })
}

// edit is text with the override for dll in its DllOverrides section, adding the section when it is missing. Lines are
// split and rejoined on the file's own ending, so what it does not touch survives byte for byte.
func edit(text, dll string, now func() int64) (string, bool) {
	eol := lineEnding(text)
	lines, changed := withOverride(strings.Split(text, eol), dll, now)
	return strings.Join(lines, eol), changed
}

// lineEnding is the file's line ending. A trailing bare CR counts as CRLF: the "\n" an appended section follows it with
// would otherwise join it into a CRLF and flip the file's ending on the next run.
func lineEnding(text string) string {
	if strings.Contains(text, "\r\n") || strings.HasSuffix(text, "\r") {
		return "\r\n"
	}
	return "\n"
}

func withOverride(lines []string, dll string, now func() int64) ([]string, bool) {
	line := fmt.Sprintf(`"%s"="native,builtin"`, dll)
	start := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(strings.ToLower(l), strings.ToLower(dllOverridesHeader)) })
	if start < 0 {
		unix := now()
		filetime := (unix + filetimeEpoch) * 10_000_000
		out := slices.Clone(lines)
		// A file ends with a newline, so its last element is empty and the blank line before a section already exists.
		if n := len(out); n > 0 && out[n-1] == "" {
			out = out[:n-1]
		}
		return append(out, "", fmt.Sprintf("%s %d", dllOverridesHeader, unix), fmt.Sprintf("#time=%x", filetime), line, ""), true
	}
	end := start + 1
	for end < len(lines) && !strings.HasPrefix(lines[end], "[") && lines[end] != "" {
		end++
	}
	for i := start + 1; i < end; i++ {
		if strings.HasPrefix(strings.ToLower(lines[i]), `"`+strings.ToLower(dll)+`"=`) {
			if lines[i] == line {
				return lines, false
			}
			out := slices.Clone(lines)
			out[i] = line
			return out, true
		}
	}
	out := slices.Clone(lines[:end])
	out = append(out, line)
	return append(out, lines[end:]...), true
}

// ZPath is the absolute host path p as the game under Wine reads it, on the Z: drive that maps the host's root.
func ZPath(p string) string {
	return `Z:` + strings.ReplaceAll(filepath.ToSlash(p), "/", `\`)
}
