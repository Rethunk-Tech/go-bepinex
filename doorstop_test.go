package bepinex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteConfigInsertsDoorstopMajorTargetKey(t *testing.T) {
	target := "/profile/BepInEx/core/BepInEx.dll"
	values := map[string]string{"targetAssembly": target, "target_assembly": target}

	got4 := rewriteConfig("targetAssembly=old.dll\n", values, nil, 4, target)
	if !strings.Contains(got4, "target_assembly="+target) {
		t.Fatalf("doorstop 4 missing snake key:\n%s", got4)
	}

	got3 := rewriteConfig("target_assembly=old.dll\n", values, nil, 3, target)
	if !strings.Contains(got3, "targetAssembly="+target) {
		t.Fatalf("doorstop 3 missing camel key:\n%s", got3)
	}
}

func TestDoorstopFilesSkipsConfigDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "doorstop_config.ini"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := DoorstopFiles(root, "winhttp.dll")
	if len(got) != 0 {
		t.Fatalf("DoorstopFiles = %q, want none", got)
	}
}
