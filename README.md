# npm-search (Bubble Tea CLI)

A minimal TUI for searching npm, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

## Install

Install from source via Go (replace REPLACEME with your GitHub user/org after pushing this repo to GitHub):

```
go install github.com/fredrikmwold/npm-search/cmd/npm-search@latest
```

Prebuilt binaries are published on GitHub Releases when you push a tag like `v0.1.0`.

## Run

```
go run ./cmd/npm-search
```

## Build

```
go build -o bin/npm-search ./cmd/npm-search
```

## Usage

- Type to enter a query.
- Press Enter to search and open results.
- Use Up/Down to navigate results. Press Enter to open the sidebar.
- Press i to install, I to install as dev dependency.
- Press Esc or Ctrl+C to quit.

## Release (maintainers)

This repo uses GoReleaser. To cut a release:

1. Set the module path in `go.mod` to your repo, e.g. `module github.com/fredrikmwold/npm-search`.
2. `.goreleaser.yaml` and workflow are already set to `fredrikmwold/npm-search`.
3. Tag a version and push:

```
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions will build and upload archives for Linux/macOS/Windows (amd64/arm64).
