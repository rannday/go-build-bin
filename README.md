# go-build-bin

`go-build-bin` is a reusable Go build tool for Go projects.

It builds deterministic release archives, writes `checksums.txt`, and targets common Windows, Linux, and macOS binaries by default.
Binaries are built to a temp file per target, then streamed into archives with bounded memory via `io.Copy`.

## Install

Pin it in a consumer repo with Go tools:

```bash
go get -tool github.com/rannday/go-build-bin@latest
```

That adds tool to consumer repo `go.mod`.

Run it from that repo with:

```bash
go tool go-build-bin -h
go tool go-build-bin -v 1.2.3
```

Pin specific version if you do not want latest:

```bash
go get -tool github.com/rannday/go-build-bin@v1.2.3
```

## Usage

```bash
go-build-bin -v VERSION [flags]
```

`-h, --help` prints help.

Default output directory is version-scoped:

```text
tmp/release/<version>
```

`--out DIR` uses exactly that directory.

`--clean` removes only the resolved output directory.

`--force` allows overwriting existing artifacts.

Default package detection:

- `--name` defaults to repo directory name
- `--main` prefers `./cmd/<name>` when that package exists
- otherwise `--main` falls back to repo root when it has runnable Go files

Generated `-ldflags` order:

1. `-s -w -buildid=` unless `--no-strip`
2. `-X <version-var>=<version>` when `--version-var` is set
3. user `--ldflags` value, appended last

Release builds also pass `-trimpath` and `-buildvcs=false` for reproducible binaries.

Archive filenames:

`<name>-<version>-<goos>-<goarch>.<format>`

Archive notes:

- Windows targets build `.exe` binaries.
- Non-Windows targets build plain binary names.
- Archive contents stay deterministic through fixed timestamps and sorted entries.
- Default targets:
  - `windows/amd64:zip`
  - `linux/amd64:tar.gz`
  - `linux/arm64:tar.gz`
  - `darwin/amd64:tar.gz`
  - `darwin/arm64:tar.gz`
- Default checksum filename: `checksums.txt`
- Default Go binary: `go`

## Example

```bash
go tool go-build-bin -v 1.2.3 --name myapp --main ./cmd/myapp --version-var github.com/rannday/myapp/internal/app.Version
```

Expected output directory:

```text
tmp/release/1.2.3
```

## Release Workflow

Typical flow:

```bash
go test ./...
go tool go-build-bin -v 1.2.3 --name myapp --main ./cmd/myapp --version-var github.com/rannday/myapp/internal/app.Version
```

Use the archives and `checksums.txt` under `tmp/release/<version>` with your own release uploader.
