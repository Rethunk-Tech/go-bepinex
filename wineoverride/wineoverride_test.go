package wineoverride

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Rethunk-Tech/go-safefs"
)

func TestEnsure(t *testing.T) {
	fixture, err := safefs.ReadFile("testdata/user.reg")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"section present":   string(fixture),
		"section absent":    strings.Replace(string(fixture), "[Software\\\\Wine\\\\DllOverrides]", "[Software\\\\Wine\\\\Other]", 1),
		"stale value":       strings.Replace(string(fixture), `"dxgi"="native,builtin"`, `"winhttp"="builtin"`, 1),
		"crlf line endings": strings.ReplaceAll(string(fixture), "\n", "\r\n"),
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			reg := filepath.Join(t.TempDir(), "user.reg")
			backup := reg + ".bak"
			if err := safefs.WriteFile(reg, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := Ensure(reg, "winhttp", backup); err != nil {
				t.Fatal(err)
			}
			once, _ := safefs.ReadFile(reg)
			if err := Ensure(reg, "winhttp", backup); err != nil {
				t.Fatal(err)
			}
			twice, _ := safefs.ReadFile(reg)
			if string(once) != string(twice) || strings.Count(string(once), `"winhttp"="native,builtin"`) != 1 {
				t.Fatalf("not idempotent or duplicated:\n%s", once)
			}
			if b, _ := safefs.ReadFile(backup); string(b) != content {
				t.Fatal("backup is not the original")
			}
			eol := "\n"
			if strings.Contains(content, "\r\n") {
				eol = "\r\n"
			}
			if strings.Contains(strings.ReplaceAll(string(once), eol, ""), "\n") {
				t.Fatal("line endings mixed")
			}
			rest := strings.Split(strings.Replace(string(once), `"winhttp"="native,builtin"`+eol, "", 1), eol)
			for l := range strings.SplitSeq(content, eol) {
				if l != "" && !strings.Contains(strings.Join(rest, eol), l) && !strings.HasPrefix(l, `"winhttp"=`) && !strings.HasPrefix(l, `[Software\\Wine\\Other]`) {
					t.Fatalf("line lost: %q", l)
				}
			}
		})
	}
}

func TestEnsureRefusesANameThatWouldEditAnotherLine(t *testing.T) {
	reg := filepath.Join(t.TempDir(), "user.reg")
	if err := safefs.WriteFile(reg, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "a\"b", "a\nb", `a\b`, "a=b"} {
		if err := Ensure(reg, bad, ""); !errors.Is(err, ErrBadName) {
			t.Errorf("Ensure(%q) = %v", bad, err)
		}
	}
	if b, _ := safefs.ReadFile(reg); string(b) != "x\n" {
		t.Fatalf("file changed: %q", b)
	}
	if err := Ensure(reg, "win http", ""); err != nil {
		t.Fatalf("space in name: %v", err)
	}
}

func TestEditAddsDllOverridesSection(t *testing.T) {
	unix := int64(1_700_000_000)
	now := func() int64 { return unix }
	filetime := (unix + filetimeEpoch) * 10_000_000
	section := fmt.Sprintf("%s %d\n#time=%x\n\"winhttp\"=\"native,builtin\"\n", dllOverridesHeader, unix, filetime)
	cases := map[string]struct{ in, want string }{
		"after trailing newline":             {"WINE REGISTRY Version 2\n", "WINE REGISTRY Version 2\n\n" + section},
		"empty file":                         {"", "\n" + section},
		"without trailing newline":           {"WINE REGISTRY Version 2", "WINE REGISTRY Version 2\n\n" + section},
		"existing section, no final newline": {dllOverridesHeader + "\n\"dxgi\"=\"native,builtin\"", dllOverridesHeader + "\n\"dxgi\"=\"native,builtin\"\n\"winhttp\"=\"native,builtin\""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, changed := edit(tc.in, "winhttp", now)
			if !changed || got != tc.want {
				t.Fatalf("changed=%v got:\n%q\nwant:\n%q", changed, got, tc.want)
			}
		})
	}
}

func TestZPath(t *testing.T) {
	if got := ZPath("/home/a b/x"); got != `Z:\home\a b\x` && filepath.Separator == '/' {
		t.Fatalf("ZPath = %q", got)
	}
}
