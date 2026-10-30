package bepinex

import (
	"strings"
	"testing"
)

// FuzzRewriteConfig feeds doorstop_config.ini contents from a pack: lines that are not settings survive, the target is
// always present once, and the line endings stay.
func FuzzRewriteConfig(f *testing.F) {
	for _, s := range []string{
		"", "[General]\r\ntarget_assembly = x\r\n[Il2Cpp]\r\ncorlib_dir = dotnet\r\n",
		"[UnityDoorstop]\ntargetAssembly=a\n[MonoBackend]\nruntimeLib=mono\\x\nconfigDir=..\\..\\etc\n",
		"=\n==\nkey\n", "target_assembly\r\r\n",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		const target = "/p/BepInEx/core/BepInEx.Preloader.dll"
		values := map[string]string{"targetAssembly": target, "target_assembly": target, "corlib_dir": "/p/dotnet"}
		resolve := func(v string) (string, bool) { return "/p/" + v, v != "" }
		out := rewriteConfig(text, values, resolve, 4, target)
		in, got := strings.Split(text, "\n"), strings.Split(out, "\n")
		if len(got) < len(in) || len(got) > len(in)+1 {
			t.Fatalf("lines %d -> %d", len(in), len(got))
		}
		if !strings.Contains(out, "="+target) {
			t.Fatalf("no target in %q", out)
		}
		for i, l := range in {
			if !strings.Contains(l, "=") && got[i] != l {
				t.Fatalf("non-setting line changed: %q -> %q", l, got[i])
			}
			if strings.HasSuffix(l, "\r") != strings.HasSuffix(got[i], "\r") && strings.Contains(l, "=") {
				t.Fatalf("line ending changed: %q -> %q", l, got[i])
			}
		}
	})
}
