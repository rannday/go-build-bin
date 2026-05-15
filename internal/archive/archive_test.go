package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAtomicZipDeterministic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bin.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o755); err != nil {
		t.Fatal(err)
	}

	out1 := filepath.Join(dir, "one.zip")
	out2 := filepath.Join(dir, "two.zip")
	items := []Item{{Name: "myapp", Path: src}}

	if err := WriteAtomic(out1, FormatZip, items); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(out2, FormatZip, items); err != nil {
		t.Fatal(err)
	}

	sum1, err := fileSum(out1)
	if err != nil {
		t.Fatal(err)
	}
	sum2, err := fileSum(out2)
	if err != nil {
		t.Fatal(err)
	}
	if sum1 != sum2 {
		t.Fatalf("zip sums differ: %s %s", sum1, sum2)
	}

	zr, err := zip.OpenReader(out1)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 1 || zr.File[0].Name != "myapp" {
		t.Fatalf("zip entries = %#v", zr.File)
	}
}

func TestWriteAtomicTarGzDeterministic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "bin.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o755); err != nil {
		t.Fatal(err)
	}

	out1 := filepath.Join(dir, "one.tar.gz")
	out2 := filepath.Join(dir, "two.tar.gz")
	items := []Item{{Name: "myapp", Path: src}}

	if err := WriteAtomic(out1, FormatTarGz, items); err != nil {
		t.Fatal(err)
	}
	if err := WriteAtomic(out2, FormatTarGz, items); err != nil {
		t.Fatal(err)
	}

	sum1, err := fileSum(out1)
	if err != nil {
		t.Fatal(err)
	}
	sum2, err := fileSum(out2)
	if err != nil {
		t.Fatal(err)
	}
	if sum1 != sum2 {
		t.Fatalf("tar sums differ: %s %s", sum1, sum2)
	}

	file, err := os.Open(out1)
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

func fileSum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
