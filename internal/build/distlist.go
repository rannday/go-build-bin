package build

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	distMu    sync.Mutex
	distCache = map[string]map[string]struct{}{}
)

func validateTargetsIfGoAvailable(targets []TargetSpec, goBinary string) error {
	if !goBinaryResolvable(goBinary) {
		return nil
	}
	return validateTargets(targets, goBinary)
}

func goBinaryResolvable(goBinary string) bool {
	if goBinary == "" || goBinary == "go" {
		return true
	}
	if _, err := exec.LookPath(goBinary); err == nil {
		return true
	}
	info, err := os.Stat(goBinary)
	return err == nil && !info.IsDir()
}

func ValidateTargetPlatform(target TargetSpec, goBinary string) error {
	if goBinary == "" {
		goBinary = "go"
	}

	platforms, err := distPlatforms(goBinary)
	if err != nil {
		return err
	}

	key := target.GOOS + "/" + target.GOARCH
	if _, ok := platforms[key]; ok {
		return nil
	}
	return fmt.Errorf("unsupported target platform %s", key)
}

func distPlatforms(goBinary string) (map[string]struct{}, error) {
	distMu.Lock()
	defer distMu.Unlock()

	if set, ok := distCache[goBinary]; ok {
		return set, nil
	}

	cmd := exec.Command(goBinary, "tool", "dist", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list target platforms: %w", err)
	}

	set := make(map[string]struct{})
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		set[line] = struct{}{}
	}
	distCache[goBinary] = set
	return set, nil
}