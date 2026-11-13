package logline

import (
	"strings"
	"testing"
)

func TestParseReadsLevelsSourcesAndStamps(t *testing.T) {
	tests := []struct {
		line string
		want Line
		ok   bool
	}{
		{"[Message:   BepInEx] BepInEx 5.4.23.5 - Lethal Company", Line{Level: Message, Source: "BepInEx", Message: "BepInEx 5.4.23.5 - Lethal Company"}, true},
		{"[Info   :Lethal Company Input Utils] Registered 3 keybinds", Line{Level: Info, Source: "Lethal Company Input Utils", Message: "Registered 3 keybinds"}, true},
		{"[Debug  :  HarmonyX] Patching void Foo::Bar()", Line{Level: Debug, Source: "HarmonyX", Message: "Patching void Foo::Bar()"}, true},
		{"[Warning:CullFactory] Portal has no renderer", Line{Level: Warning, Source: "CullFactory", Message: "Portal has no renderer"}, true},
		{"[Fatal  :   BepInEx] Error occurred starting the game", Line{Level: Fatal, Source: "BepInEx", Message: "Error occurred starting the game"}, true},
		{"[14:35:39.1344735] [Info   :SlendermanMod] Registered spawn rate", Line{Stamp: "14:35:39.1344735", Level: Info, Source: "SlendermanMod", Message: "Registered spawn rate"}, true},
		{"  at Foo.Bar () [0x00000] in <abc>:0", Line{}, false},
		{"[Trace  : X] unknown level", Line{}, false},
	}
	for _, tt := range tests {
		got, ok := Parse(tt.line)
		if ok != tt.ok || got != tt.want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v, %v", tt.line, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSplitStampLeavesUnstampedLinesWhole(t *testing.T) {
	if s, rest := SplitStamp("[Info   : A] b"); s != "" || rest != "[Info   : A] b" {
		t.Errorf("unstamped: %q, %q", s, rest)
	}
	if s, rest := SplitStamp("[0000000000000007] [Error  : A] b"); s != "0000000000000007" || rest != "[Error  : A] b" {
		t.Errorf("stamped: %q, %q", s, rest)
	}
}

func TestFormatWritesLinesBackInBepInExsFormat(t *testing.T) {
	for _, line := range []string{
		"[Error  :   BepInEx] Could not load [A 1.0.0] because it has missing dependencies: B",
		"[Warning:CullFactory] Portal has no renderer",
		"[14:35:39.1344735] [Info   :SlendermanMod] Registered spawn rate",
	} {
		l, ok := Parse(line)
		if !ok {
			t.Fatalf("Parse(%q) failed", line)
		}
		if got := Format(l); got != line {
			t.Errorf("Format = %q, want %q", got, line)
		}
	}
}

func TestParseStripsTrailingCR(t *testing.T) {
	l, ok := Parse("[Info   :   BepInEx] hello\r")
	if !ok {
		t.Fatal("Parse rejected a Windows log line")
	}
	if l.Message != "hello" {
		t.Fatalf("Message = %q", l.Message)
	}
}

func TestFormatNeverEmitsASecondLine(t *testing.T) {
	got := Format(Line{Stamp: "a\nb", Level: Info, Source: "x\r\ny", Message: "hi\n[Error  :  Forged] boom"})
	if strings.ContainsAny(got, "\r\n") {
		t.Errorf("Format = %q", got)
	}
}
