package build

import (
	"bytes"
	"errors"
	"testing"
)

func TestParseArgsShortFlags(t *testing.T) {
	var help bytes.Buffer
	opts, err := ParseArgsWithUsage([]string{
		"-v", "1.2.3",
		"-n", "myapp",
		"-m", "./cmd/myapp",
		"-o", "dist",
		"-c",
		"-f",
		"--no-strip",
		"--verbose",
		"--go", "custom-go",
		"--checksum-name", "sums.txt",
		"--ldflags", "-buildid=abc",
		"-t", "linux/arm64",
	}, &help)
	if err != nil {
		t.Fatal(err)
	}

	if opts.Version != "1.2.3" || opts.Name != "myapp" || opts.MainPackage != "./cmd/myapp" {
		t.Fatalf("unexpected opts: %#v", opts)
	}
	if opts.OutDir != "dist" || !opts.Clean || !opts.Force || !opts.NoStrip || !opts.Verbose {
		t.Fatalf("unexpected output opts: %#v", opts)
	}
	if opts.GoBinary != "custom-go" || opts.ChecksumName != "sums.txt" || opts.Ldflags != "-buildid=abc" {
		t.Fatalf("unexpected build opts: %#v", opts)
	}
	if len(opts.Targets) != 1 || opts.Targets[0].GOOS != "linux" || opts.Targets[0].GOARCH != "arm64" {
		t.Fatalf("targets = %#v", opts.Targets)
	}
}

func TestParseArgsLongFlags(t *testing.T) {
	var help bytes.Buffer
	opts, err := ParseArgsWithUsage([]string{
		"--version", "1.2.3",
		"--name", "myapp",
		"--main", "./cmd/myapp",
		"--out", "dist",
		"--clean",
		"--force",
		"--no-strip",
		"--verbose",
		"--go", "custom-go",
		"--checksum-name", "sums.txt",
		"--ldflags", "-buildid=abc",
		"--target", "darwin/arm64",
	}, &help)
	if err != nil {
		t.Fatal(err)
	}

	if opts.Version != "1.2.3" || opts.Name != "myapp" || opts.MainPackage != "./cmd/myapp" {
		t.Fatalf("unexpected opts: %#v", opts)
	}
	if opts.OutDir != "dist" || !opts.Clean || !opts.Force || !opts.NoStrip || !opts.Verbose {
		t.Fatalf("unexpected output opts: %#v", opts)
	}
	if opts.GoBinary != "custom-go" || opts.ChecksumName != "sums.txt" || opts.Ldflags != "-buildid=abc" {
		t.Fatalf("unexpected build opts: %#v", opts)
	}
	if len(opts.Targets) != 1 || opts.Targets[0].GOOS != "darwin" || opts.Targets[0].GOARCH != "arm64" {
		t.Fatalf("targets = %#v", opts.Targets)
	}
}

func TestParseArgsHelp(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}} {
		var help bytes.Buffer
		_, err := ParseArgsWithUsage(args, &help)
		if !errors.Is(err, ErrHelp) {
			t.Fatalf("expected ErrHelp, got %v", err)
		}
		assertHelpOutput(t, help.String())
	}
}

func TestParseArgsMissingLongFlagArgumentNormalizesName(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--version"}, "flag needs an argument: --version"},
		{[]string{"--version-var"}, "flag needs an argument: --version-var"},
		{[]string{"--main"}, "flag needs an argument: --main"},
		{[]string{"--out"}, "flag needs an argument: --out"},
		{[]string{"--target"}, "flag needs an argument: --target"},
		{[]string{"--ldflags"}, "flag needs an argument: --ldflags"},
		{[]string{"--go"}, "flag needs an argument: --go"},
		{[]string{"--checksum-name"}, "flag needs an argument: --checksum-name"},
	}

	for _, tt := range tests {
		var help bytes.Buffer
		_, err := ParseArgsWithUsage(tt.args, &help)
		if err == nil {
			t.Fatalf("ParseArgsWithUsage(%v) expected error", tt.args)
		}
		if err.Error() != tt.want {
			t.Fatalf("ParseArgsWithUsage(%v) error = %q, want %q", tt.args, err.Error(), tt.want)
		}
		if help.Len() != 0 {
			t.Fatalf("parse error should not print usage, got %q", help.String())
		}
	}
}

func TestParseArgsMissingShortFlagArgumentKeepsName(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-v"}, "flag needs an argument: -v"},
		{[]string{"-n"}, "flag needs an argument: -n"},
		{[]string{"-m"}, "flag needs an argument: -m"},
		{[]string{"-o"}, "flag needs an argument: -o"},
		{[]string{"-t"}, "flag needs an argument: -t"},
	}

	for _, tt := range tests {
		var help bytes.Buffer
		_, err := ParseArgsWithUsage(tt.args, &help)
		if err == nil {
			t.Fatalf("ParseArgsWithUsage(%v) expected error", tt.args)
		}
		if err.Error() != tt.want {
			t.Fatalf("ParseArgsWithUsage(%v) error = %q, want %q", tt.args, err.Error(), tt.want)
		}
		if help.Len() != 0 {
			t.Fatalf("parse error should not print usage, got %q", help.String())
		}
	}
}

func TestParseArgsRequiresVersion(t *testing.T) {
	var help bytes.Buffer
	_, err := ParseArgsWithUsage([]string{"--name", "myapp"}, &help)
	if err == nil {
		t.Fatal("expected error")
	}
}
