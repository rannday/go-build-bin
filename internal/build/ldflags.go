package build

import (
	"fmt"
	"strings"
)

func BuildLdflags(version, versionVar, extra string, strip bool) string {
	var parts []string
	if strip {
		parts = append(parts, "-s", "-w", "-buildid=")
	}
	if versionVar != "" {
		parts = append(parts, "-X", fmt.Sprintf("%s=%s", versionVar, version))
	}
	if extra != "" {
		parts = append(parts, extra)
	}
	return strings.Join(parts, " ")
}
