# npm-search

[![Go Reference](https://pkg.go.dev/badge/github.com/fredrikmwold/npm-search.svg)](https://pkg.go.dev/github.com/fredrikmwold/npm-search)
[![Release](https://img.shields.io/github/v/release/FredrikMWold/npm-search?sort=semver)](https://github.com/FredrikMWold/npm-search/releases)

**A minimal, keyboard-first TUI for npm** built with [Bubble Tea](https://github.com/charmbracelet/bubbletea). Search, inspect, install, and manage/update your project's npm packages without leaving your terminal.

![Demo](./demo.gif)


<details>
	<summary><strong>Quick keys</strong></summary>

| Context | Key | Action |
|---|---|---|
| Input | `Enter` | Run search for current query |
| Results | `↑`/`↓` | Move selection |
| Results | `Enter` | Toggle details sidebar for selected package |
| Results | `i` | Install selected package |
| Results | `I` | Install as dev dependency |
| Results | `u` | Update selected package to latest (if installed) |
| Anywhere | `Tab` | Toggle focus between input and results |
| Anywhere | `Esc` | Clear input and show your project packages |
| Anywhere | `Ctrl+C` | Quit |

> Tip: The help footer updates based on what you can do at the moment.

</details>

## Features

- 🔎 Fast npm search from the terminal
- 🧰 Manage and update your project's npm packages
- 📊 Results show version, weekly downloads, license, and author
- 📚 Details sidebar with description and quick links (homepage, repo, npm)
- ⌨️ One-key install (i), dev install (I), and update (u) when installed
- 🧠 Auto-detects npm, pnpm, yarn, and bun via lockfiles
- 🧩 Responsive layout with a toggleable sidebar

## Install

Install with Go:

```sh
go install github.com/fredrikmwold/npm-search/cmd/npm-search@latest
```

Or download a prebuilt binary from the Releases page and place it on your PATH:

- https://github.com/FredrikMWold/npm-search/releases