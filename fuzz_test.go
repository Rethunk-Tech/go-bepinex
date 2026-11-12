package bepinex

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func FuzzRewriteConfig(f *testing.F) {
	f.Add("[UnityDoorstop]\nenabled=true\ntargetAssembly=BepInEx/core/BepInEx.Preloader.dll\n", 3)
	f.Add("[UnityDoorstop]\r\nenabled=true\r\ntargetAssembly=BepInEx\\core\\BepInEx.Preloader.dll\r\n", 3)
	f.Add("[General]\r\nenabled=true\r\ntarget_assembly=BepInEx/core/BepInEx.Preloader.dll\r\n", 4)
	f.Add("targetAssembly=BepInEx/core/BepInEx.Preloader.dll", 3)
	f.Add("", 4)
	f.Add("runtimeLib=dotnet\nconfigDir=BepInEx/config\n", 4)
	f.Fuzz(func(t *testing.T, text string, doorstopMajor int) {
		const target = "T"
		values := map[string]string{"targetAssembly": target, "target_assembly": target}
		out := rewriteConfig(text, values, func(string) (string, bool) { return "", false }, doorstopMajor, target)
		if !strings.Contains(out, target) {
			t.Fatalf("rewritten config lost target %q", target)
		}
		n := strings.Count(text, "\n") + 1
		if len(out) > len(text)+n+len("target_assembly=")+len(target)+1 {
			t.Fatalf("rewrite grew past bound: %d -> %d", len(text), len(out))
		}
		wantKey := "targetAssembly"
		if doorstopMajor >= 4 {
			wantKey = "target_assembly"
		}
		if !strings.Contains(out, "targetAssembly="+target) && !strings.Contains(out, "target_assembly="+target) && !strings.Contains(out, wantKey+"="+target) {
			t.Fatalf("missing target assignment in %q", out)
		}
	})
}

func FuzzUnityVersion(f *testing.F) {
	f.Add([]byte("2022.3.21f1"))
	f.Add([]byte("2019.4.40f1"))
	f.Add([]byte("2021.3.16f1"))
	f.Add([]byte("5.6.7f1"))
	f.Add([]byte(""))
	f.Add([]byte("not a version"))
	f.Fuzz(func(t *testing.T, head []byte) {
		if len(head) > unityHeaderBytes {
			head = head[:unityHeaderBytes]
		}
		dir := t.TempDir()
		data := filepath.Join(dir, "Game_Data")
		if err := os.MkdirAll(data, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(data, "globalgamemanagers"), head, 0o600); err != nil {
			t.Fatal(err)
		}
		ver := UnityVersion(dir)
		if ver == "" {
			return
		}
		if !unityVersionCaptured.MatchString(ver) {
			t.Fatalf("UnityVersion returned %q", ver)
		}
	})
}

var unityVersionCaptured = regexp.MustCompile(`^\d{4}\.\d+\.\d+$`)

func FuzzCheckUnityZip(f *testing.F) {
	f.Add([]byte("PK"))
	f.Add([]byte{})
	f.Add(zipBytes("mscorlib.dll"))
	f.Add(zipBytes("../escape.dll"))
	f.Add(zipBytes("ok/../ok.dll"))
	f.Fuzz(func(t *testing.T, b []byte) {
		err := checkUnityZip(b)
		if err != nil {
			return
		}
		zr, zerr := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if zerr != nil {
			t.Fatal(zerr)
		}
		var total uint64
		for _, file := range zr.File {
			if !filepath.IsLocal(filepath.FromSlash(file.Name)) {
				t.Fatalf("accepted non-local entry %q", file.Name)
			}
			total += file.UncompressedSize64
		}
		if total > maxUnityLibsUnpacked {
			t.Fatal("accepted zip that unpacks too large")
		}
	})
}

func zipBytes(name string) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	fw, err := w.Create(name)
	if err == nil {
		_, _ = fw.Write([]byte("x"))
	}
	_ = w.Close()
	return buf.Bytes()
}
