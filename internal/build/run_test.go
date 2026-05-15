package build

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBuildsArchivesAndChecksum(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdDir := filepath.Join(root, "cmd", "myapp")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)

	res, err := Run(Options{
		Version: "1.2.3",
		Name:    "myapp",
	})
	if err != nil {
		t.Fatal(err)
	}

	if res.ChecksumPath == "" {
		t.Fatal("missing checksum path")
	}
	if len(res.Archives) != len(DefaultTargets()) {
		t.Fatalf("archives = %d", len(res.Archives))
	}

	for _, path := range res.Archives {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}

	checksumData, err := os.ReadFile(res.ChecksumPath)
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(strings.TrimSpace(string(checksumData)), "\n") + 1; lines != len(DefaultTargets()) {
		t.Fatalf("checksum lines = %d", lines)
	}

	zipPath := filepath.Join(res.OutputDir, "myapp-1.2.3-windows-amd64.zip")
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 1 || zr.File[0].Name != "myapp.exe" {
		t.Fatalf("zip files = %#v", zr.File)
	}

	tarPath := filepath.Join(res.OutputDir, "myapp-1.2.3-linux-amd64.tar.gz")
	file, err := os.Open(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatal(err)
	}
	if hdr.Name != "myapp" {
		t.Fatalf("tar entry = %q", hdr.Name)
	}
}
