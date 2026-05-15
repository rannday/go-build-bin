package checksum

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rannday/go-build-bin/internal/atomicfile"
)

type Entry struct {
	Name string
	Sum  string
}

func SumFile(path string) (string, error) {
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

func WriteAtomic(dir, name string, entries []Entry) (string, error) {
	path := filepath.Join(dir, name)
	entries = append([]Entry(nil), entries...)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	var b strings.Builder
	for _, entry := range entries {
		fmt.Fprintf(&b, "%s  %s\n", entry.Sum, entry.Name)
	}

	if err := atomicfile.Write(path, func(tmpPath string) error {
		return os.WriteFile(tmpPath, []byte(b.String()), 0o644)
	}); err != nil {
		return "", err
	}

	return path, nil
}
