package logline

import (
	"strings"
	"testing"
)

// FuzzParse feeds log lines a mod wrote: any text parses without panic, and a line it reads survives Format and Parse.
func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"", "[Info   :   BepInEx] hi", "[1] [Error  : x] y", "[] [Debug:]", "[Info", "[Warning:]]] ]", "[a] [b] [Info :",
		"[Fatal  :\xff] \x00", "[Message:\n] x",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, line string) {
		l, ok := Parse(line)
		if !ok {
			return
		}
		// A bare header whose message opens with "[" is ambiguous with a stamped line once formatted.
		if l.Stamp == "" && strings.HasPrefix(l.Message, "[") {
			return
		}
		again, ok := Parse(Format(l))
		if !ok || again != l {
			t.Fatalf("%q -> %+v -> %q -> %+v, %v", line, l, Format(l), again, ok)
		}
	})
}
