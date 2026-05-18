package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/rannday/go-build-bin/internal/build"
)

func main() {
	opts, err := build.ParseArgs(os.Args[1:])
	if err != nil {
		if errors.Is(err, build.ErrHelp) {
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		build.PrintUsage(os.Stderr)
		os.Exit(1)
	}

	if _, err := build.Run(opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
