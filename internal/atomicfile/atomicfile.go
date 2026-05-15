package atomicfile

import (
	"os"
	"path/filepath"
)

func Write(path string, fn func(tmpPath string) error) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Remove(tmpPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := fn(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return nil
}
