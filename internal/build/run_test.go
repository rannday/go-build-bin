package build

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rannday/go-build-bin/internal/archive"
)

func linuxAMD64Target() TargetSpec {
	return TargetSpec{GOOS: "linux", GOARCH: "amd64", Format: archive.FormatTarGz}
}

func setupTestModule(t *testing.T) string {
	t.Helper()

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
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	return root
}

func TestRunBuildsArchivesAndChecksum(t *testing.T) {
	setupTestModule(t)

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
	if res.OutputDirRel != filepath.Join("dist", "1.2.3") {
		t.Fatalf("OutputDirRel = %q", res.OutputDirRel)
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

func TestRunRejectsNonEmptyOutputDir(t *testing.T) {
	setupTestModule(t)

	opts := Options{
		Version: "1.2.3",
		Name:    "myapp",
		Targets: []TargetSpec{linuxAMD64Target()},
	}
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(opts); err == nil {
		t.Fatal("expected non-empty output directory error")
	}
}

func TestRunCleanAllowsRebuild(t *testing.T) {
	setupTestModule(t)

	opts := Options{
		Version: "1.2.3",
		Name:    "myapp",
		Targets: []TargetSpec{linuxAMD64Target()},
	}
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}

	opts.Clean = true
	if _, err := Run(opts); err != nil {
		t.Fatal(err)
	}
}

func TestRunHonorsExplicitOutputDirExactly(t *testing.T) {
	root := setupTestModule(t)

	opts := Options{
		Version: "1.2.3",
		Name:    "myapp",
		OutDir:  "custom-out",
		Targets: []TargetSpec{linuxAMD64Target()},
	}
	res, err := Run(opts)
	if err != nil {
		t.Fatal(err)
	}

	if res.OutputDirRel != "custom-out" {
		t.Fatalf("OutputDirRel = %q", res.OutputDirRel)
	}
	if res.OutputDir != filepath.Join(root, "custom-out") {
		t.Fatalf("OutputDir = %q", res.OutputDir)
	}
	if strings.Contains(res.OutputDir, "1.2.3") {
		t.Fatalf("OutputDir should not include version: %q", res.OutputDir)
	}
}

func TestDirIsEmptyIgnoresBuildTempDirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".build-abc123"), 0o755); err != nil {
		t.Fatal(err)
	}

	empty, err := dirIsEmpty(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !empty {
		t.Fatal("expected temp build dir to be ignored")
	}

	if err := os.WriteFile(filepath.Join(dir, "artifact.zip"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty, err = dirIsEmpty(dir)
	if err != nil {
		t.Fatal(err)
	}
	if empty {
		t.Fatal("expected directory with artifact to be non-empty")
	}
}
