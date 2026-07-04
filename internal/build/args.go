package build

import (
	"errors"
	"flag"
	"io"
	"os"
	"strings"
)

type Options struct {
	Version      string
	Name         string
	MainPackage  string
	VersionVar   string
	OutDir       string
	Clean        bool
	Force        bool
	NoStrip      bool
	Verbose      bool
	GoBinary     string
	ChecksumName string
	Ldflags      string
	Targets      []TargetSpec
}

type targetList []TargetSpec

func (tl *targetList) String() string {
	if tl == nil {
		return ""
	}
	out := make([]string, 0, len(*tl))
	for _, target := range *tl {
		out = append(out, target.String())
	}
	return strings.Join(out, ",")
}

func (tl *targetList) Set(raw string) error {
	target, err := ParseTarget(raw)
	if err != nil {
		return err
	}
	*tl = append(*tl, target)
	return nil
}

func ParseArgs(args []string) (Options, error) {
	return ParseArgsWithUsage(args, os.Stdout)
}

func ParseArgsWithUsage(args []string, usageOut io.Writer) (Options, error) {
	if usageOut == nil {
		usageOut = io.Discard
	}

	var opts Options
	var help bool
	var targets targetList

	fs := newFlagSet(usageOut)
	fs.StringVar(&opts.Version, "version", "", "release version (required)")
	fs.StringVar(&opts.Version, "v", "", "release version (required)")
	fs.StringVar(&opts.Name, "name", "", "binary/release name (default: repo directory)")
	fs.StringVar(&opts.Name, "n", "", "binary/release name (default: repo directory)")
	fs.StringVar(&opts.MainPackage, "main", "", "main package (default: ./cmd/<name>, then repo root)")
	fs.StringVar(&opts.MainPackage, "m", "", "main package (default: ./cmd/<name>, then repo root)")
	fs.StringVar(&opts.VersionVar, "version-var", "", "Go symbol set with -ldflags -X")
	fs.StringVar(&opts.OutDir, "out", "", "output directory (default: tmp/release/<version>)")
	fs.StringVar(&opts.OutDir, "o", "", "output directory (default: tmp/release/<version>)")
	fs.BoolVar(&opts.Clean, "clean", false, "remove output directory before building")
	fs.BoolVar(&opts.Clean, "c", false, "remove output directory before building")
	fs.BoolVar(&opts.Force, "force", false, "overwrite existing artifacts")
	fs.BoolVar(&opts.Force, "f", false, "overwrite existing artifacts")
	fs.BoolVar(&opts.NoStrip, "no-strip", false, "do not add -s -w linker flags")
	fs.BoolVar(&opts.Verbose, "verbose", false, "print build commands and Go output")
	fs.Var(&targets, "target", "build target, repeatable: GOOS/GOARCH[:zip|tar.gz]")
	fs.Var(&targets, "t", "build target, repeatable: GOOS/GOARCH[:zip|tar.gz]")
	fs.StringVar(&opts.Ldflags, "ldflags", "", "additional linker flags")
	fs.StringVar(&opts.GoBinary, "go", "go", "Go command to run (default: go)")
	fs.StringVar(&opts.ChecksumName, "checksum-name", DefaultChecksumName, "checksum file name (default: "+DefaultChecksumName+")")
	fs.BoolVar(&help, "help", false, "display this help and exit")
	fs.BoolVar(&help, "h", false, "display this help and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			PrintUsage(usageOut)
			return Options{}, ErrHelp
		}
		return Options{}, normalizeFlagError(err, longFlagNamesFrom(fs))
	}

	if help {
		PrintUsage(usageOut)
		return Options{}, ErrHelp
	}

	if err := ValidateVersion(opts.Version); err != nil {
		return Options{}, err
	}

	if len(targets) == 0 {
		opts.Targets = DefaultTargets()
	} else {
		opts.Targets = append([]TargetSpec(nil), targets...)
	}

	if err := validateTargetsIfGoAvailable(opts.Targets, opts.GoBinary); err != nil {
		return Options{}, err
	}

	return opts, nil
}

func newFlagSet(usageOut io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("go-build-bin", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return fs
}

func longFlagNamesFrom(fs *flag.FlagSet) []string {
	var names []string
	seen := make(map[string]struct{})
	fs.VisitAll(func(f *flag.Flag) {
		if len(f.Name) == 1 {
			return
		}
		if _, ok := seen[f.Name]; ok {
			return
		}
		seen[f.Name] = struct{}{}
		names = append(names, f.Name)
	})
	return names
}

func normalizeFlagError(err error, longFlagNames []string) error {
	if err == nil {
		return nil
	}

	msg := err.Error()
	for _, name := range longFlagNames {
		msg = strings.ReplaceAll(msg, ": -"+name, ": --"+name)
		msg = strings.ReplaceAll(msg, "flag provided but not defined: -"+name, "flag provided but not defined: --"+name)
	}
	return errors.New(msg)
}
