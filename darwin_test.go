package bepinex

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestPlacementForDarwinLibdoorstopInBracketPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Game [Steam]")
	if err := os.MkdirAll(filepath.Join(root, "doorstop_libs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"run_bepinex.sh", "libdoorstop.dylib", "doorstop_libs/libdoorstop_x64.dylib"} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := PlacementFor(root, "darwin")
	want := []string{"run_bepinex.sh", "libdoorstop.dylib", "doorstop_libs/libdoorstop_x64.dylib"}
	if !slices.Equal(got.Files, want) {
		t.Fatalf("Files = %q, want %q", got.Files, want)
	}
}
