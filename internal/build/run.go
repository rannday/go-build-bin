package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rannday/go-build-bin/internal/archive"
	"github.com/rannday/go-build-bin/internal/checksum"
)

type Result struct {
	RepoRoot      string
	Name          string
	Version       string
	MainPackage   string
	OutputDir     string
	ArtifactDir   string
	Archives      []string
	ChecksumPath  string
	ChecksumsPath string
}

func Run(opts Options) (Result, error) {
	repoRoot, err := FindRepoRoot()
	if err != nil {
		return Result{}, err
	}

	if err := ValidateVersion(opts.Version); err != nil {
		return Result{}, err
	}

	if opts.GoBinary == "" {
		opts.GoBinary = "go"
	}
	if opts.ChecksumName == "" {
		opts.ChecksumName = "checksums.txt"
	}

	name := opts.Name
	if name == "" {
		name = filepath.Base(repoRoot)
	}

	mainPkg := opts.MainPackage
	if mainPkg == "" {
		mainPkg, err = ResolveDefaultMain(repoRoot, name)
		if err != nil {
			return Result{}, err
		}
	}

	outAbs, _, err := ResolveOutputDir(repoRoot, opts.Version, opts.OutDir)
	if err != nil {
		return Result{}, err
	}

	targets := opts.Targets
	if len(targets) == 0 {
		targets = DefaultTargets()
	}

	if err := ValidateUniqueArchiveNames(name, opts.Version, targets); err != nil {
		return Result{}, err
	}

	if opts.Clean {
		if err := os.RemoveAll(outAbs); err != nil {
			return Result{}, err
		}
	}

	if err := os.MkdirAll(outAbs, 0o755); err != nil {
		return Result{}, err
	}

	if !opts.Force {
		empty, err := dirIsEmpty(outAbs)
		if err != nil {
			return Result{}, err
		}
		if !empty {
			return Result{}, fmt.Errorf("output directory not empty: %s", outAbs)
		}
	}

	archives := make([]string, 0, len(targets))
	checksumEntries := make([]checksum.Entry, 0, len(targets))

	for _, target := range targets {
		archivePath := filepath.Join(outAbs, ArchiveName(name, opts.Version, target))
		buildDir, err := os.MkdirTemp(outAbs, ".build-*")
		if err != nil {
			return Result{}, err
		}

		err = func() error {
			defer os.RemoveAll(buildDir)

			binaryPath := filepath.Join(buildDir, BinaryName(name, target.GOOS))
			if err := buildTarget(repoRoot, mainPkg, opts, target, binaryPath); err != nil {
				return err
			}

			if err := archive.WriteAtomic(archivePath, target.Format, []archive.Item{
				{Name: BinaryName(name, target.GOOS), Path: binaryPath},
			}); err != nil {
				return err
			}

			sum, err := checksum.SumFile(archivePath)
			if err != nil {
				return err
			}

			archives = append(archives, archivePath)
			checksumEntries = append(checksumEntries, checksum.Entry{
				Name: filepath.Base(archivePath),
				Sum:  sum,
			})
			return nil
		}()
		if err != nil {
			return Result{}, err
		}
	}

	checksumPath, err := checksum.WriteAtomic(outAbs, opts.ChecksumName, checksumEntries)
	if err != nil {
		return Result{}, err
	}

	return Result{
		RepoRoot:      repoRoot,
		Name:          name,
		Version:       opts.Version,
		MainPackage:   mainPkg,
		OutputDir:     outAbs,
		ArtifactDir:   outAbs,
		Archives:      archives,
		ChecksumPath:  checksumPath,
		ChecksumsPath: checksumPath,
	}, nil
}

func buildTarget(repoRoot, mainPkg string, opts Options, target TargetSpec, binaryPath string) error {
	ldflags := BuildLdflags(opts.Version, opts.VersionVar, opts.Ldflags, !opts.NoStrip)
	args := []string{"build", "-trimpath", "-o", binaryPath}
	if ldflags != "" {
		args = append(args, "-ldflags", ldflags)
	}
	args = append(args, mainPkg)

	cmd := exec.Command(opts.GoBinary, args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		"GOOS="+target.GOOS,
		"GOARCH="+target.GOARCH,
		"CGO_ENABLED=0",
	)

	if opts.Verbose {
		fmt.Fprintf(os.Stdout, "%s\n", strings.Join(cmd.Args, " "))
	}

	output, err := cmd.CombinedOutput()
	if opts.Verbose && len(output) > 0 {
		_, _ = os.Stdout.Write(output)
	}
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
		}
		return err
	}

	return nil
}

func dirIsEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}
