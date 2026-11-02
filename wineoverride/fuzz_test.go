package wineoverride

import (
	"slices"
	"strings"
	"testing"
)

// FuzzEdit feeds user.reg contents: the edit adds or sets exactly one override line and leaves every other line, and
// the file's line ending, as it was.
func FuzzEdit(f *testing.F) {
	for _, s := range []string{
		"", "\n", "WINE REGISTRY Version 2\n\n[Software\\\\Wine\\\\DllOverrides] 1\n#time=1\n\"dxgi\"=\"native\"\n",
		"[Software\\\\Wine\\\\DllOverrides] 1\r\n\"winhttp\"=\"builtin\"\r\n\r\n[x]\r\n",
		"a\r\nb\nc", "0\r", "\r\n\n", "[software\\\\wine\\\\dlloverrides]\n\"WINHTTP\"=\"x\"\n",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		now := func() int64 { return 1 }
		once, changed := edit(text, "winhttp", now)
		if !changed {
			if once != text {
				t.Fatal("unchanged but rewritten")
			}
			return
		}
		twice, again := edit(once, "winhttp", now)
		if again || twice != once {
			t.Fatalf("not idempotent:\n%q\n%q", once, twice)
		}
		eol := lineEnding(text)
		have := strings.Split(once, eol)
		for l := range strings.SplitSeq(text, eol) {
			if l == "" || strings.HasPrefix(strings.ToLower(l), `"winhttp"=`) {
				continue
			}
			i := slices.Index(have, l)
			if i < 0 {
				t.Fatalf("line lost: %q", l)
			}
			have = have[i+1:]
		}
	})
}
