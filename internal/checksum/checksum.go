package checksum

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func WriteAtomic(dir, name string, archives []string) (string, error) {
	sorted := append([]string(nil), archives...)
	sort.Slice(sorted, func(i, j int) bool {
		return filepath.Base(sorted[i]) < filepath.Base(sorted[j])
	})

	var buf bytes.Buffer
	for _, archive := range sorted {
		sum, err := SHA256File(archive)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&buf, "%s  %s\n", sum, filepath.Base(archive))
	}

	path := filepath.Join(dir, name)
	tmp, err := os.CreateTemp(dir, filepath.Base(name)+".tmp-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		_ = os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", err
	}
	return path, nil
}
