package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteCreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	err := Write(path, func(tmpPath string) error {
		return os.WriteFile(tmpPath, []byte("hello"), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteRemovesTempOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	var gotTmpPath string

	err := Write(path, func(tmpPath string) error {
		gotTmpPath = tmpPath
		if err := os.WriteFile(tmpPath, []byte("partial"), 0o644); err != nil {
			return err
		}
		return os.ErrInvalid
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dest exists: %v", err)
	}
	if _, err := os.Stat(gotTmpPath); !os.IsNotExist(err) {
		t.Fatalf("temp exists: %v", err)
	}
}

func TestWriteReplacesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")

	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := Write(path, func(tmpPath string) error {
		return os.WriteFile(tmpPath, []byte("new"), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("got %q", got)
	}
}
