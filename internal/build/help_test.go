package build

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintUsage(t *testing.T) {
	var buf bytes.Buffer
	PrintUsage(&buf)
	assertHelpOutput(t, buf.String())
}

func assertHelpOutput(t *testing.T, got string) {
	t.Helper()

	mustContain := []string{
		"Usage:",
		"go-build-bin [options] -v <version>",
		"Options:",
		"-v, --version <version>",
		"--target <target>",
		"-h, --help",
		"Default targets:",
		"Output:",
		"dist/<version>",
		"<name>-<version>-<goos>-<goarch>.<format>",
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Fatalf("help missing %q\n%s", want, got)
		}
	}

	for _, absent := range []string{"NAME", "SYNOPSIS", "DESCRIPTION", "EXAMPLES", "-V", "--version-info"} {
		if strings.Contains(got, absent) {
			t.Fatalf("help has %q\n%s", absent, got)
		}
	}

	for _, target := range DefaultTargetStrings() {
		if !strings.Contains(got, target) {
			t.Fatalf("help missing target %q\n%s", target, got)
		}
	}

	if strings.Index(got, "-v, --version <version>") > strings.Index(got, "--verbose") {
		t.Fatalf("version flag after verbose\n%s", got)
	}
	if strings.Index(got, "--verbose") > strings.Index(got, "-h, --help") {
		t.Fatalf("help flag not at bottom\n%s", got)
	}
}
