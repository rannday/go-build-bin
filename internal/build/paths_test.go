package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOutputDir(t *testing.T) {
	root := filepath.Join("C:", "repo")

	abs, display, err := ResolveOutputDir(root, "0.1.2", "")
	if err != nil {
		t.Fatal(err)
	}
	if abs != filepath.Join(root, "tmp", "release", "0.1.2") {
		t.Fatalf("abs = %q", abs)
	}
	if display != filepath.Join("tmp", "release", "0.1.2") {
		t.Fatalf("display = %q", display)
	}

	abs, display, err = ResolveOutputDir(root, "0.1.2", filepath.Join("dist", "out"))
	if err != nil {
		t.Fatal(err)
	}
	if abs != filepath.Join(root, "dist", "out") || display != filepath.Join("dist", "out") {
		t.Fatalf("explicit out wrong: %s %s", abs, display)
	}
}

func TestResolveDefaultMainPrefersCmdDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdDir := filepath.Join(root, "cmd", "myapp")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	mainPkg, err := ResolveDefaultMain(root, "myapp")
	if err != nil {
		t.Fatal(err)
	}
	if mainPkg != "./cmd/myapp" {
		t.Fatalf("mainPkg = %q", mainPkg)
	}
}
