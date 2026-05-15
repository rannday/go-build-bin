package build

import (
	"fmt"
	"io"
)

func PrintUsage(w io.Writer) {
	targets := DefaultTargetStrings()

	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  go-build-bin [options] -v <version>")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	printHelpOption(w, "-v, --version <version>", "release version (required)")
	printHelpOption(w, "-n, --name <name>", "binary/release name (default: repo directory)")
	printHelpOption(w, "-m, --main <package>", "main package (default: ./cmd/<name>, then repo root)")
	printHelpOption(w, "--version-var <symbol>", "Go symbol set with -ldflags -X")
	printHelpOption(w, "-o, --out <dir>", "output directory (default: tmp/release/<version>)")
	printHelpOption(w, "-c, --clean", "remove output directory before building")
	printHelpOption(w, "-f, --force", "overwrite existing artifacts")
	printHelpOption(w, "-t, --target <target>", "build target, repeatable: GOOS/GOARCH[:zip|tar.gz]")
	printHelpOption(w, "--ldflags <value>", "additional linker flags")
	printHelpOption(w, "--no-strip", "do not add -s -w linker flags")
	printHelpOption(w, "--go <path>", "Go command to run (default: go)")
	printHelpOption(w, "--checksum-name <name>", "checksum file name (default: checksums.txt)")
	printHelpOption(w, "--verbose", "print build commands and Go output")
	fmt.Fprintln(w)
	printHelpOption(w, "-h, --help", "display this help and exit")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Default targets:")
	fmt.Fprintf(w, "  %s, %s, %s,\n", targets[0], targets[1], targets[2])
	fmt.Fprintf(w, "  %s, %s\n", targets[3], targets[4])
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Output:")
	fmt.Fprintln(w, "  archives: <name>-<version>-<goos>-<goarch>.<format>")
	fmt.Fprintln(w, "  checksum: checksums.txt")
}

func printHelpOption(w io.Writer, name, desc string) {
	fmt.Fprintf(w, "  %-29s %s\n", name, desc)
}
