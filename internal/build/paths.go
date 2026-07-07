package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		} else if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find repo root")
		}
		dir = parent
	}
}

func ResolveDefaultMain(repoRoot, name string) (string, error) {
	cmdDir := filepath.Join(repoRoot, "cmd", name)
	ok, err := packageDirHasGoFiles(cmdDir)
	if err != nil {
		return "", err
	}
	if ok {
		return "./cmd/" + name, nil
	}

	ok, err = packageDirHasGoFiles(repoRoot)
	if err != nil {
		return "", err
	}
	if ok {
		return ".", nil
	}

	return "", fmt.Errorf("could not find runnable Go package for %q", name)
}

func ResolveOutputDir(repoRoot, version, out string) (string, string, error) {
	if out != "" {
		if filepath.IsAbs(out) {
			clean := filepath.Clean(out)
			return clean, clean, nil
		}
		display := filepath.Clean(out)
		return filepath.Clean(filepath.Join(repoRoot, display)), display, nil
	}

	display := filepath.Join("dist", version)
	return filepath.Join(repoRoot, display), display, nil
}

func packageDirHasGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) == ".go" && !strings.HasSuffix(entry.Name(), "_test.go") {
			return true, nil
		}
	}
	return false, nil
}
