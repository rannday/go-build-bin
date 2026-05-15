package build

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rannday/go-build-bin/internal/archive"
	"github.com/rannday/go-build-bin/internal/checksum"
)

var (
	ErrHelp        = errors.New("help requested")
	ErrVersion     = errors.New("version requested")
	versionPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

type Options struct {
	Version      string
	Name         string
	Main         string
	VersionVar   string
	OutDir       string
	Flat         bool
	Clean        bool
	Force        bool
	NoStrip      bool
	Verbose      bool
	GoBinary     string
	ChecksumName string
	Ldflags      string
	Targets      []TargetSpec
}

type Result struct {
	ArtifactDir   string
	Archives      []string
	ChecksumsPath string
}

type TargetSpec struct {
	GOOS   string
	GOARCH string
	Format string
}

type targetList []string

func (t *targetList) String() string {
	return strings.Join(*t, ",")
}

func (t *targetList) Set(value string) error {
	*t = append(*t, value)
	return nil
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

func ParseArgs(args []string) (Options, error) {
	var opts Options
	var targets targetList
	var help bool
	var versionInfo bool

	fs := newFlagSet()
	fs.StringVar(&opts.Version, "version", "", "release version")
	fs.StringVar(&opts.Version, "v", "", "release version")
	fs.StringVar(&opts.Name, "name", "", "binary name")
	fs.StringVar(&opts.Name, "n", "", "binary name")
	fs.StringVar(&opts.Main, "main", "", "main package")
	fs.StringVar(&opts.Main, "m", "", "main package")
	fs.StringVar(&opts.VersionVar, "version-var", "", "ldflags version symbol")
	fs.StringVar(&opts.OutDir, "out", "", "output directory")
	fs.StringVar(&opts.OutDir, "o", "", "output directory")
	fs.BoolVar(&opts.Clean, "clean", false, "remove output directory before building")
	fs.BoolVar(&opts.Clean, "c", false, "remove output directory before building")
	fs.BoolVar(&opts.Force, "force", false, "overwrite existing artifacts")
	fs.BoolVar(&opts.Force, "f", false, "overwrite existing artifacts")
	fs.BoolVar(&opts.Flat, "flat", false, "write artifacts directly in tmp/release")
	fs.BoolVar(&opts.NoStrip, "no-strip", false, "disable -s -w")
	fs.BoolVar(&opts.Verbose, "verbose", false, "print commands and Go output")
	fs.StringVar(&opts.GoBinary, "go", "go", "go binary")
	fs.StringVar(&opts.ChecksumName, "checksum-name", "checksums.txt", "checksum file name")
	fs.StringVar(&opts.Ldflags, "ldflags", "", "additional ldflags")
	fs.Var(&targets, "target", "GOOS/GOARCH[:FORMAT]")
	fs.Var(&targets, "t", "GOOS/GOARCH[:FORMAT]")
	fs.BoolVar(&help, "help", false, "print help")
	fs.BoolVar(&help, "h", false, "print help")
	fs.BoolVar(&versionInfo, "version-info", false, "output version information and exit")
	fs.BoolVar(&versionInfo, "V", false, "output version information and exit")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Options{}, ErrHelp
		}
		return Options{}, err
	}
	if help {
		fs.Usage()
		return Options{}, ErrHelp
	}
	if versionInfo {
		return Options{}, ErrVersion
	}
	if opts.Version == "" {
		return Options{}, errors.New("missing required version")
	}
	if err := ValidateVersion(opts.Version); err != nil {
		return Options{}, err
	}
	if opts.GoBinary == "" {
		opts.GoBinary = "go"
	}
	if opts.ChecksumName == "" {
		opts.ChecksumName = "checksums.txt"
	}
	if len(targets) == 0 {
		opts.Targets = DefaultTargets()
	} else {
		for _, raw := range targets {
			target, err := ParseTarget(raw)
			if err != nil {
				return Options{}, err
			}
			opts.Targets = append(opts.Targets, target)
		}
	}
	return opts, nil
}

func newFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("go-build-bin", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() { PrintUsage(os.Stdout) }
	return fs
}

func PrintUsage(w io.Writer) {
	fmt.Fprint(w, `Usage:
  go-build-bin [options] -v <version>

Options:
  -v, --version <version>      release version (required)
  -n, --name <name>            binary/release name (default: repo directory)
  -m, --main <package>         main package (default: ./cmd/<name>, then repo root)
      --version-var <symbol>   Go symbol set with -ldflags -X
  -o, --out <dir>              output directory (default: tmp/release/<version>)
  -c, --clean                  remove output directory before building
  -f, --force                  overwrite existing artifacts
      --flat                   write to tmp/release instead of tmp/release/<version>
  -t, --target <target>        build target, repeatable: GOOS/GOARCH[:zip|tar.gz]
      --ldflags <value>        additional linker flags
      --no-strip               do not add -s -w linker flags
      --go <path>              Go command to run (default: go)
      --checksum-name <name>    checksum file name (default: checksums.txt)
      --verbose                print build commands and Go output

  -h, --help                   display this help and exit
  -V, --version-info           output version information and exit

Default targets:
  windows/amd64:zip, linux/amd64:tar.gz, linux/arm64:tar.gz,
  darwin/amd64:tar.gz, darwin/arm64:tar.gz

Output:
  archives: <name>-<version>-<goos>-<goarch>.<format>
  checksum: checksums.txt
`)
}

func Run(ctx context.Context, opts Options) (Result, error) {
	if opts.Version == "" {
		return Result{}, errors.New("missing required version")
	}
	if err := ValidateVersion(opts.Version); err != nil {
		return Result{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	repoRoot, err := FindRepoRoot()
	if err != nil {
		return Result{}, err
	}

	name := opts.Name
	if name == "" {
		name = filepath.Base(repoRoot)
	}

	mainPkg := opts.Main
	if mainPkg == "" {
		mainPkg, err = ResolveDefaultMain(repoRoot, name)
		if err != nil {
			return Result{}, err
		}
	}

	outAbs, outDisplay, err := ResolveOutputDir(repoRoot, opts.Version, opts.Flat, opts.OutDir)
	if err != nil {
		return Result{}, err
	}

	if err := ValidateUniqueArchiveNames(name, opts.Version, opts.Targets); err != nil {
		return Result{}, err
	}

	if opts.Clean {
		if err := os.RemoveAll(outAbs); err != nil {
			return Result{}, fmt.Errorf("clean output dir: %w", err)
		}
	}

	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return Result{}, fmt.Errorf("create output dir: %w", err)
	}

	if !opts.Clean && !opts.Force {
		entries, err := os.ReadDir(outAbs)
		if err != nil {
			return Result{}, fmt.Errorf("read output dir: %w", err)
		}
		if len(entries) > 0 {
			return Result{}, fmt.Errorf("output directory is not empty: %s", outAbs)
		}
	}

	stageRoot, err := os.MkdirTemp(filepath.Dir(outAbs), "go-build-bin-*")
	if err != nil {
		return Result{}, fmt.Errorf("create staging dir: %w", err)
	}
	defer os.RemoveAll(stageRoot)

	var archives []string
	for _, target := range opts.Targets {
		archiveName := ArchiveName(name, opts.Version, target)
		finalPath := filepath.Join(outAbs, archiveName)

		stagedArchive, err := buildTarget(ctx, repoRoot, stageRoot, name, mainPkg, opts, target)
		if err != nil {
			return Result{}, err
		}

		if err := finalizeArchive(stagedArchive, finalPath, opts.Force || opts.Clean); err != nil {
			return Result{}, err
		}
		archives = append(archives, finalPath)
	}

	checksumsPath, err := checksum.WriteAtomic(outAbs, opts.ChecksumName, archives)
	if err != nil {
		return Result{}, err
	}

	return Result{
		ArtifactDir:   outDisplay,
		Archives:      displayPaths(repoRoot, archives),
		ChecksumsPath: displayPath(repoRoot, checksumsPath),
	}, nil
}

func buildTarget(ctx context.Context, repoRoot, stageRoot, name, mainPkg string, opts Options, target TargetSpec) (string, error) {
	stageDir := filepath.Join(stageRoot, target.GOOS+"-"+target.GOARCH)
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return "", fmt.Errorf("create staging dir: %w", err)
	}

	binaryName := BinaryName(name, target.GOOS)
	binaryPath := filepath.Join(stageDir, binaryName)
	ldflags := BuildLdflags(opts.Version, opts.VersionVar, opts.NoStrip, opts.Ldflags)

	args := []string{
		"build",
		"-trimpath",
		"-buildvcs=false",
	}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, "-o", binaryPath, mainPkg)

	cmd := exec.CommandContext(ctx, opts.GoBinary, args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+target.GOOS,
		"GOARCH="+target.GOARCH,
	)

	if opts.Verbose {
		fmt.Fprintf(os.Stderr, "%s %s\n", opts.GoBinary, strings.Join(args, " "))
		fmt.Fprintf(os.Stderr, "CGO_ENABLED=0 GOOS=%s GOARCH=%s\n", target.GOOS, target.GOARCH)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			_, _ = os.Stderr.Write(output)
			if output[len(output)-1] != '\n' {
				_, _ = os.Stderr.WriteString("\n")
			}
		}
		return "", fmt.Errorf("go build failed for %s/%s: %w", target.GOOS, target.GOARCH, err)
	}
	if opts.Verbose && len(output) > 0 {
		_, _ = os.Stderr.Write(output)
		if output[len(output)-1] != '\n' {
			_, _ = os.Stderr.WriteString("\n")
		}
	}

	archivePath := filepath.Join(stageDir, ArchiveName(name, opts.Version, target))
	item := archive.Item{Source: binaryPath, Name: binaryName, Mode: 0o755}
	if err := archive.WriteAtomic(archivePath, target.Format, []archive.Item{item}); err != nil {
		return "", err
	}
	return archivePath, nil
}

func finalizeArchive(stagedArchive, finalPath string, overwrite bool) error {
	if overwrite {
		if err := os.Remove(finalPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(stagedArchive, finalPath); err != nil {
		return err
	}
	return nil
}

func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		modPath := filepath.Join(dir, "go.mod")
		_, statErr := os.Stat(modPath)
		switch {
		case statErr == nil:
			return dir, nil
		case errors.Is(statErr, os.ErrNotExist):
		default:
			return "", fmt.Errorf("stat %s: %w", modPath, statErr)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate repo root containing go.mod")
		}
		dir = parent
	}
}

func ResolveDefaultMain(repoRoot, name string) (string, error) {
	cmdDir := filepath.Join(repoRoot, "cmd", name)
	hasGo, err := packageDirHasGoFiles(cmdDir)
	if err != nil {
		return "", err
	}
	if hasGo {
		return filepath.ToSlash(filepath.Join(".", "cmd", name)), nil
	}
	hasGo, err = packageDirHasGoFiles(repoRoot)
	if err != nil {
		return "", err
	}
	if hasGo {
		return ".", nil
	}
	return "", fmt.Errorf("default main package not found: ./cmd/%s does not exist and repo root has no runnable Go files; pass --main", name)
}

func ResolveOutputDir(repoRoot, version string, flat bool, out string) (string, string, error) {
	if out != "" {
		if filepath.IsAbs(out) {
			return filepath.Clean(out), filepath.Clean(out), nil
		}
		display := filepath.Clean(out)
		return filepath.Clean(filepath.Join(repoRoot, display)), display, nil
	}

	display := filepath.Join("tmp", "release")
	if !flat {
		display = filepath.Join(display, version)
	}
	return filepath.Join(repoRoot, display), display, nil
}

func ValidateVersion(version string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("invalid version %q: use letters, numbers, dots, underscores, and hyphens only", version)
	}
	return nil
}

func ParseTarget(raw string) (TargetSpec, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return TargetSpec{}, fmt.Errorf("invalid target %q", raw)
	}

	spec := raw
	format := ""
	if strings.Count(raw, ":") > 1 {
		return TargetSpec{}, fmt.Errorf("invalid target %q", raw)
	}
	if idx := strings.Index(raw, ":"); idx >= 0 {
		spec = raw[:idx]
		format = raw[idx+1:]
	}

	parts := strings.Split(spec, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return TargetSpec{}, fmt.Errorf("invalid target %q", raw)
	}

	if format == "" {
		format = defaultFormat(parts[0])
	}
	if format != archive.FormatZip && format != archive.FormatTarGz {
		return TargetSpec{}, fmt.Errorf("unsupported archive format: %s", format)
	}

	return TargetSpec{
		GOOS:   parts[0],
		GOARCH: parts[1],
		Format: format,
	}, nil
}

func DefaultBinaryName(name, goos string) string {
	if goos == "windows" {
		return name + ".exe"
	}
	return name
}

func BinaryName(name, goos string) string {
	return DefaultBinaryName(name, goos)
}

func ArchiveName(name, version string, target TargetSpec) string {
	ext := target.Format
	return fmt.Sprintf("%s-%s-%s-%s.%s", name, version, target.GOOS, target.GOARCH, ext)
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

func BuildLdflags(version, versionVar string, noStrip bool, userLdflags string) string {
	var parts []string
	if !noStrip {
		parts = append(parts, "-s", "-w")
	}
	if versionVar != "" {
		parts = append(parts, "-X", fmt.Sprintf("%s=%s", versionVar, version))
	}
	if strings.TrimSpace(userLdflags) != "" {
		parts = append(parts, strings.TrimSpace(userLdflags))
	}
	return strings.Join(parts, " ")
}

func defaultFormat(goos string) string {
	if goos == "windows" {
		return archive.FormatZip
	}
	return archive.FormatTarGz
}

func packageDirHasGoFiles(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("read dir %s: %w", dir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".go") && !strings.HasSuffix(entry.Name(), "_test.go") {
			return true, nil
		}
	}
	return false, nil
}

func displayPath(repoRoot, abs string) string {
	if repoRoot == "" {
		return abs
	}
	rel, err := filepath.Rel(repoRoot, abs)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(rel)
	}
	return abs
}

func displayPaths(repoRoot string, absPaths []string) []string {
	out := make([]string, 0, len(absPaths))
	for _, abs := range absPaths {
		out = append(out, displayPath(repoRoot, abs))
	}
	return out
}
