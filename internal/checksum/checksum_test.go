package checksum

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSumFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "artifact.zip")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	sum, err := SumFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(sum) != 64 {
		t.Fatalf("sum length = %d", len(sum))
	}

	sumAgain, err := SumFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if sum != sumAgain {
		t.Fatalf("sums differ: %s %s", sum, sumAgain)
	}
}

func TestWriteAtomicSortsEntries(t *testing.T) {
	dir := t.TempDir()
	path, err := WriteAtomic(dir, "checksums.txt", []Entry{
		{Name: "z.tar.gz", Sum: "bbb"},
		{Name: "a.zip", Sum: "aaa"},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(data))
	want := "aaa  a.zip\nbbb  z.tar.gz"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
