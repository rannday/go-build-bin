package build

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rannday/go-build-bin/internal/archive"
	"github.com/rannday/go-build-bin/internal/checksum"
)

func TestValidateVersion(t *testing.T) {
	if err := ValidateVersion("0.1.2"); err != nil {
		t.Fatalf("valid version rejected: %v", err)
	}
	if err := ValidateVersion("v0.1.2"); err != nil {
		t.Fatalf("valid version rejected: %v", err)
	}
	if err := ValidateVersion("bad version"); err == nil {
		t.Fatal("invalid version accepted")
	}
}

func TestParseTarget(t *testing.T) {
	t.Run("default format", func(t *testing.T) {
		got, err := ParseTarget("linux/amd64")
		if err != nil {
			t.Fatal(err)
		}
		if got.GOOS != "linux" || got.GOARCH != "amd64" || got.Format != "tar.gz" {
			t.Fatalf("unexpected target: %#v", got)
		}
	})
	t.Run("explicit zip", func(t *testing.T) {
		got, err := ParseTarget("windows/amd64:zip")
		if err != nil {
			t.Fatal(err)
		}
		if got.Format != "zip" {
			t.Fatalf("unexpected format: %#v", got)
		}
	})
	t.Run("invalid", func(t *testing.T) {
		if _, err := ParseTarget("bad"); err == nil {
			t.Fatal("invalid target accepted")
		}
	})
	t.Run("unsupported format", func(t *testing.T) {
		if _, err := ParseTarget("linux/amd64:rar"); err == nil || !strings.Contains(err.Error(), "unsupported archive format") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParseArgsShortFlags(t *testing.T) {
	opts, err := ParseArgs([]string{
		"-v", "1.2.3",
		"-n", "myapp",
		"-m", "./cmd/myapp",
		"--version-var", "github.com/rannday/myapp/internal/app.Version",
		"-o", "dist",
		"-c",
		"-f",
		"--flat",
		"--no-strip",
		"--verbose",
		"--go", "custom-go",
		"--checksum-name", "sums.txt",
		"--ldflags", "-buildid=abc",
		"-t", "linux/arm64",
	})
	if err != nil {
		t.Fatal(err)
	}

	if opts.Version != "1.2.3" || opts.Name != "myapp" || opts.Main != "./cmd/myapp" {
		t.Fatalf("unexpected project opts: %#v", opts)
	}
	if opts.VersionVar != "github.com/rannday/myapp/internal/app.Version" {
		t.Fatalf("version var = %q", opts.VersionVar)
	}
	if opts.OutDir != "dist" || !opts.Clean || !opts.Force || !opts.Flat || !opts.NoStrip || !opts.Verbose {
		t.Fatalf("unexpected output opts: %#v", opts)
	}
	if opts.GoBinary != "custom-go" || opts.ChecksumName != "sums.txt" || opts.Ldflags != "-buildid=abc" {
		t.Fatalf("unexpected build opts: %#v", opts)
	}
	if len(opts.Targets) != 1 {
		t.Fatalf("targets = %#v", opts.Targets)
	}
	if opts.Targets[0].GOOS != "linux" || opts.Targets[0].GOARCH != "arm64" || opts.Targets[0].Format != "tar.gz" {
		t.Fatalf("unexpected target: %#v", opts.Targets[0])
	}
}

func TestParseArgsLongFlags(t *testing.T) {
	opts, err := ParseArgs([]string{
		"--version", "2.3.4",
		"--name", "myapp",
		"--main", "./cmd/myapp",
		"--version-var", "github.com/rannday/myapp/internal/app.Version",
		"--out", "dist",
		"--clean",
		"--force",
		"--flat",
		"--no-strip",
		"--verbose",
		"--go", "custom-go",
		"--checksum-name", "sums.txt",
		"--ldflags", "-buildid=abc",
		"--target", "darwin/arm64",
	})
	if err != nil {
		t.Fatal(err)
	}

	if opts.Version != "2.3.4" || opts.Name != "myapp" || opts.Main != "./cmd/myapp" {
		t.Fatalf("unexpected project opts: %#v", opts)
	}
	if opts.VersionVar != "github.com/rannday/myapp/internal/app.Version" {
		t.Fatalf("version var = %q", opts.VersionVar)
	}
	if opts.OutDir != "dist" || !opts.Clean || !opts.Force || !opts.Flat || !opts.NoStrip || !opts.Verbose {
		t.Fatalf("unexpected output opts: %#v", opts)
	}
	if opts.GoBinary != "custom-go" || opts.ChecksumName != "sums.txt" || opts.Ldflags != "-buildid=abc" {
		t.Fatalf("unexpected build opts: %#v", opts)
	}
	if len(opts.Targets) != 1 {
		t.Fatalf("targets = %#v", opts.Targets)
	}
	if opts.Targets[0].GOOS != "darwin" || opts.Targets[0].GOARCH != "arm64" || opts.Targets[0].Format != "tar.gz" {
		t.Fatalf("unexpected target: %#v", opts.Targets[0])
	}
}

func TestParseArgsHelp(t *testing.T) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	_, err = ParseArgs([]string{"-h"})
	if err == nil {
		t.Fatal("help should return error")
	}
	if !errors.Is(err, ErrHelp) {
		t.Fatalf("unexpected help error: %v", err)
	}

	_ = w.Close()
	output := <-done
	if !strings.Contains(output, "Usage: go-build-bin -v VERSION [flags]") {
		t.Fatalf("help output missing usage line: %q", output)
	}
	if !strings.Contains(output, "--version-var SYMBOL") || !strings.Contains(output, "--checksum-name NAME") {
		t.Fatalf("help output missing flag lines: %q", output)
	}
}

func TestDefaultTargets(t *testing.T) {
	got := DefaultTargets()
	if len(got) != 5 {
		t.Fatalf("got %d targets", len(got))
	}
	if got[0].GOOS != "windows" || got[0].GOARCH != "amd64" || got[0].Format != "zip" {
		t.Fatalf("unexpected first target: %#v", got[0])
	}
}

func TestResolveOutputDir(t *testing.T) {
	root := filepath.Join("C:", "repo")
	abs, display, err := ResolveOutputDir(root, "0.1.2", false, "")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, "tmp", "release", "0.1.2"); abs != want {
		t.Fatalf("abs = %s, want %s", abs, want)
	}
	if display != filepath.Join("tmp", "release", "0.1.2") {
		t.Fatalf("display = %s", display)
	}

	abs, display, err = ResolveOutputDir(root, "0.1.2", true, "")
	if err != nil {
		t.Fatal(err)
	}
	if display != filepath.Join("tmp", "release") {
		t.Fatalf("display = %s", display)
	}
	if abs != filepath.Join(root, "tmp", "release") {
		t.Fatalf("abs = %s", abs)
	}

	abs, display, err = ResolveOutputDir(root, "0.1.2", false, filepath.Join("dist", "out"))
	if err != nil {
		t.Fatal(err)
	}
	if abs != filepath.Join(root, "dist", "out") || display != filepath.Join("dist", "out") {
		t.Fatalf("explicit out wrong: %s %s", abs, display)
	}
}

func TestArchiveName(t *testing.T) {
	got := ArchiveName("oaklog", "0.1.2", TargetSpec{GOOS: "linux", GOARCH: "amd64", Format: "tar.gz"})
	if got != "oaklog-0.1.2-linux-amd64.tar.gz" {
		t.Fatalf("got %s", got)
	}
}

func TestValidateUniqueArchiveNamesRejectsExactDuplicate(t *testing.T) {
	targets := []TargetSpec{
		{GOOS: "linux", GOARCH: "amd64", Format: archive.FormatTarGz},
		{GOOS: "linux", GOARCH: "amd64", Format: archive.FormatTarGz},
	}

	err := ValidateUniqueArchiveNames("myapp", "1.2.3", targets)
	if err == nil {
		t.Fatal("expected duplicate archive error")
	}
	if !strings.Contains(err.Error(), "duplicate target output: myapp-1.2.3-linux-amd64.tar.gz") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateUniqueArchiveNamesRejectsEquivalentDuplicate(t *testing.T) {
	first, err := ParseTarget("linux/amd64")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseTarget("linux/amd64:tar.gz")
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateUniqueArchiveNames("myapp", "1.2.3", []TargetSpec{first, second})
	if err == nil {
		t.Fatal("expected duplicate archive error")
	}
}

func TestValidateUniqueArchiveNamesAllowsDistinctTargets(t *testing.T) {
	targets := []TargetSpec{
		{GOOS: "linux", GOARCH: "amd64", Format: archive.FormatTarGz},
		{GOOS: "linux", GOARCH: "arm64", Format: archive.FormatTarGz},
		{GOOS: "windows", GOARCH: "amd64", Format: archive.FormatZip},
	}

	if err := ValidateUniqueArchiveNames("myapp", "1.2.3", targets); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildLdflags(t *testing.T) {
	if got := BuildLdflags("0.1.2", "", false, ""); got != "-s -w" {
		t.Fatalf("got %q", got)
	}
	if got := BuildLdflags("0.1.2", "", true, ""); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := BuildLdflags("0.1.2", "example.com/mod.Version", false, ""); got != "-s -w -X example.com/mod.Version=0.1.2" {
		t.Fatalf("got %q", got)
	}
	if got := BuildLdflags("0.1.2", "example.com/mod.Version", false, "-buildid=abc"); got != "-s -w -X example.com/mod.Version=0.1.2 -buildid=abc" {
		t.Fatalf("got %q", got)
	}
}

func TestChecksumsOnlyIncludeCreatedArchives(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(a, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}

	sumPath, err := runChecksum(dir, "checksums.txt", []string{a})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(sumPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("a.txt")) || bytes.Contains(data, []byte("b.txt")) {
		t.Fatalf("unexpected checksums content: %s", data)
	}
}

func TestIntegrationMiniModule(t *testing.T) {
	root := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/testapp\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmdDir := filepath.Join(root, "cmd", "testapp")
	if err := os.MkdirAll(cmdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	mainGo := []byte("package main\nfunc main() {}\n")
	if err := os.WriteFile(filepath.Join(cmdDir, "main.go"), mainGo, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	format := "tar.gz"
	if runtime.GOOS == "windows" {
		format = "zip"
	}

	result, err := Run(context.Background(), Options{
		Version:      "0.1.2",
		Name:         "testapp",
		Main:         "./cmd/testapp",
		GoBinary:     "go",
		ChecksumName: "checksums.txt",
		Targets: []TargetSpec{{
			GOOS:   runtime.GOOS,
			GOARCH: runtime.GOARCH,
			Format: format,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Archives) != 1 {
		t.Fatalf("archives = %v", result.Archives)
	}
	if _, err := os.Stat(filepath.Join(root, "tmp", "release", "0.1.2")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "tmp", "release", "0.1.2", "testapp-0.1.2-"+runtime.GOOS+"-"+runtime.GOARCH+"."+format)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "tmp", "release", "0.1.2", "checksums.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestArchiveDeterministic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bin")
	if err := os.WriteFile(src, []byte("abc"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, format := range []string{"zip", "tar.gz"} {
		a := filepath.Join(dir, format+"-1")
		b := filepath.Join(dir, format+"-2")
		item := []archive.Item{{Source: src, Name: "bin", Mode: 0o755}}
		if err := archive.WriteAtomic(a, format, item); err != nil {
			t.Fatal(err)
		}
		if err := archive.WriteAtomic(b, format, item); err != nil {
			t.Fatal(err)
		}
		left, err := os.ReadFile(a)
		if err != nil {
			t.Fatal(err)
		}
		right, err := os.ReadFile(b)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(left, right) {
			t.Fatalf("%s archive not deterministic", format)
		}
	}
}

func runChecksum(dir, name string, archives []string) (string, error) {
	return checksum.WriteAtomic(dir, name, archives)
}

func TestArchiveContents(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bin")
	if err := os.WriteFile(src, []byte("abc"), 0o755); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(dir, "out.zip")
	if err := archive.WriteAtomic(zipPath, "zip", []archive.Item{{Source: src, Name: "bin", Mode: 0o755}}); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(zr.File) != 1 || zr.File[0].Name != "bin" {
		t.Fatalf("bad zip contents: %#v", zr.File)
	}
	_ = zr.Close()

	tgzPath := filepath.Join(dir, "out.tar.gz")
	if err := archive.WriteAtomic(tgzPath, "tar.gz", []archive.Item{{Source: src, Name: "bin", Mode: 0o755}}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(tgzPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	hdr, err := tr.Next()
	if err != nil {
		t.Fatal(err)
	}
	if hdr.Name != "bin" {
		t.Fatalf("bad tar contents: %s", hdr.Name)
	}
}
