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
		"[Message:   BepInEx] BepInEx 5.4.23.5 - Lethal Company",
		"[Info   :Lethal Company Input Utils] Registered 3 keybinds",
		"[Debug  :  HarmonyX] Patching void Foo::Bar()",
		"[Warning:CullFactory] Portal has no renderer",
		"[Fatal  :   BepInEx] Error occurred starting the game",
		"[14:35:39.1344735] [Info   :SlendermanMod] Registered spawn rate",
		"  at Foo.Bar () [0x00000] in <abc>:0",
		"[Trace  : X] unknown level",
		"[Info   :   BepInEx] hello\r",
		"[Error  :   BepInEx] Could not load [A 1.0.0] because it has missing dependencies: B",
		"[0000000000000007] [Error  : A] b",
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
