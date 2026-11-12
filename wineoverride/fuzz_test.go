package wineoverride

import (
	"fmt"
	"strings"
	"testing"
)

func FuzzEdit(f *testing.F) {
	f.Add("WINE REGISTRY Version 2\n\n[Software\\\\Wine\\\\DllOverrides] 1\n#time=1\n\"winhttp\"=\"native,builtin\"\n", "winhttp")
	f.Add("WINE REGISTRY Version 2\r\n\r\n[Software\\\\Wine\\\\DllOverrides] 1\r\n#time=1\r\n", "winhttp")
	f.Add("WINE REGISTRY Version 2\n", "version")
	f.Add("WINE REGISTRY Version 2\r", "winhttp")
	f.Add("[Software\\\\Wine\\\\DllOverrides]\n\"version\"=\"builtin\"\n", "version")
	f.Add("", "winhttp")
	f.Fuzz(func(t *testing.T, text, dll string) {
		if !validName(dll) {
			return
		}
		now := func() int64 { return 0 }
		out, changed := edit(text, dll, now)
		line := fmt.Sprintf(`"%s"="native,builtin"`, dll)
		if !strings.Contains(out, line) {
			t.Fatalf("missing override %q", line)
		}
		if !changed && out != text {
			t.Fatalf("reported unchanged but bytes differ")
		}
		bound := len(text) + len(dllOverridesHeader) + len(line) + 80
		if len(out) > bound {
			t.Fatalf("edit grew past bound: %d -> %d", len(text), len(out))
		}
		again, againChanged := edit(out, dll, now)
		if againChanged {
			t.Fatal("second edit reported a change")
		}
		if again != out {
			t.Fatal("second edit was not idempotent")
		}
	})
}
