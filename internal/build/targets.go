package build

import (
	"fmt"
	"strings"

	"github.com/rannday/go-build-bin/internal/archive"
)

type TargetSpec struct {
	GOOS   string
	GOARCH string
	Format archive.Format
}

func (t TargetSpec) String() string {
	return fmt.Sprintf("%s/%s:%s", t.GOOS, t.GOARCH, t.Format)
}

func DefaultTargets() []TargetSpec {
	return []TargetSpec{
		{GOOS: "windows", GOARCH: "amd64", Format: archive.FormatZip},
		{GOOS: "linux", GOARCH: "amd64", Format: archive.FormatTarGz},
		{GOOS: "linux", GOARCH: "arm64", Format: archive.FormatTarGz},
		{GOOS: "darwin", GOARCH: "amd64", Format: archive.FormatTarGz},
		{GOOS: "darwin", GOARCH: "arm64", Format: archive.FormatTarGz},
	}
}

func DefaultTargetStrings() []string {
	targets := DefaultTargets()
	out := make([]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, target.String())
	}
	return out
}

func ParseTarget(raw string) (TargetSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return TargetSpec{}, fmt.Errorf("invalid target %q", raw)
	}

	base, format, hasFormat := strings.Cut(raw, ":")
	goos, goarch, ok := strings.Cut(base, "/")
	if !ok || goos == "" || goarch == "" {
		return TargetSpec{}, fmt.Errorf("invalid target %q", raw)
	}

	target := TargetSpec{GOOS: goos, GOARCH: goarch, Format: defaultFormat(goos)}
	if hasFormat {
		switch archive.Format(strings.ToLower(format)) {
		case archive.FormatZip, archive.FormatTarGz:
			target.Format = archive.Format(strings.ToLower(format))
		default:
			return TargetSpec{}, fmt.Errorf("invalid target format %q", format)
		}
	}

	return target, nil
}

func defaultFormat(goos string) archive.Format {
	if goos == "windows" {
		return archive.FormatZip
	}
	return archive.FormatTarGz
}

func ValidateUniqueArchiveNames(name, version string, targets []TargetSpec) error {
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		archiveName := ArchiveName(name, version, target)
		if _, ok := seen[archiveName]; ok {
			return fmt.Errorf("duplicate target output: %s", archiveName)
		}
		seen[archiveName] = struct{}{}
	}
	return nil
}

func ArchiveName(name, version string, target TargetSpec) string {
	return fmt.Sprintf("%s-%s-%s-%s.%s", name, version, target.GOOS, target.GOARCH, target.Format)
}

func BinaryName(name, goos string) string {
	if goos == "windows" {
		return name + ".exe"
	}
	return name
}
