package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/rannday/go-build-bin/internal/build"
)

var version = "v0.1.1"

func main() {
	opts, err := build.ParseArgs(os.Args[1:])
	if err != nil {
		switch {
		case errors.Is(err, build.ErrHelp):
			return
		case errors.Is(err, build.ErrVersion):
			fmt.Printf("go-build-bin %s\n", version)
			return
		default:
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	result, err := build.Run(context.Background(), opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Artifact directory: %s\n", result.ArtifactDir)
	fmt.Println("Created archives:")
	for _, path := range result.Archives {
		fmt.Println(path)
	}
	fmt.Println(result.ChecksumsPath)
}
