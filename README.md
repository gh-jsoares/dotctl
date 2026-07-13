# dotctl

[![CI](https://github.com/gh-jsoares/dotctl/actions/workflows/ci.yml/badge.svg)](https://github.com/gh-jsoares/dotctl/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/gh-jsoares/dotctl)](https://github.com/gh-jsoares/dotctl/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/gh-jsoares/dotctl)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A developer environment orchestrator for macOS. Manages context switching between work and personal environments, bootstraps fresh machines, and keeps your system in sync.

> **Note:** This project was vibe coded with [Claude](https://claude.ai) (Anthropic). The entire codebase was generated through conversational AI pair programming.

![demo](https://raw.githubusercontent.com/gh-jsoares/dotctl/assets/demo.gif)

## The Problem

If you use one laptop for both work and personal projects, tools leak state across environments:

- AWS credentials from work pollute personal projects
- Kubernetes contexts from multiple clusters share one kubeconfig
- Docker auth tokens from work registries conflict with Docker Hub
- Git commits go out with the wrong identity
- Work-only CLIs (awscreds, internal tools) write to shared paths

## The Solution

dotctl provides **context switching** that isolates these concerns at the filesystem level, plus orchestration to bootstrap and maintain your full environment.

```bash
ctx work       # symlinks flip, env vars change, secrets resolve
ctx personal   # everything switches back
```

## Architecture

dotctl is the orchestration layer in a stack of specialized tools:

```
dotctl              → orchestrates everything, owns context switching
nix-darwin          → machine config, packages, system settings
GNU Stow            → symlinks dotfiles into place
mise                → language/tool runtime versions
1Password CLI       → secrets retrieval
```

Each tool has a single responsibility. dotctl doesn't replace any of them — it coordinates them.

## Quick Start

### Fresh machine (from nothing)

```bash
curl -sSf https://raw.githubusercontent.com/gh-jsoares/dotctl/main/install.sh | bash
dotctl bootstrap
```

### Existing machine (already have repos)

```bash
make install                    # from the dotctl repo
dotctl ctx default personal
dotctl sync
```

## Shell Completions

```sh
# bash (add to ~/.bashrc)
eval "$(dotctl completion bash)"

# zsh (add to ~/.zshrc, before compinit)
eval "$(dotctl completion zsh)"

# fish (add to ~/.config/fish/config.fish)
dotctl completion fish | source
```

See [docs/completions.md](docs/completions.md) for static generation and caching strategies.

## Prompt Integration

dotctl exports environment variables you can use in your shell prompt (starship, p10k, etc.):

| Variable | Description | Example |
|----------|-------------|---------|
| `DOTCTL_CONTEXT` | Active context name | `personal` |
| `DOTCTL_CONTEXT_ICON` | Icon from context TOML `[prompt] icon` | `🏠` |

Starship example:

```toml
[custom.dotctl_ctx]
command = "echo ${DOTCTL_CONTEXT_ICON}${DOTCTL_CONTEXT}"
when = '[ -n "$DOTCTL_CONTEXT" ]'
style = "bold purple"
```

## Documentation

- [Configuration](docs/configuration.md) — config.toml and context definitions
- [Bootstrap](docs/bootstrap.md) — fresh machine setup flow
- [Context Switching](docs/context-switching.md) — how isolation works
- [Shell Integration](docs/shell-integration.md) — shell init, guards, chdir hooks
- [Shell Completions](docs/completions.md) — static cached generation for faster startup
- [Plugins](docs/plugins.md) — extend sync/bootstrap/doctor with custom scripts
- [Commands](docs/commands.md) — full command reference
- [Dotfiles Repo](docs/dotfiles-repo.md) — how to structure your dotfiles for dotctl

## Installation

### Homebrew (recommended)

```bash
brew install gh-jsoares/tap/dotctl
```

### From release

```bash
curl -sSf https://raw.githubusercontent.com/gh-jsoares/dotctl/main/install.sh | bash
```

The install script detects your OS/arch, downloads the binary, and places it in `/usr/local/bin`.

### From source

```bash
git clone git@github.com:gh-jsoares/dotctl.git
cd dotctl
make install  # builds and copies to ~/.local/bin/
```

### Updating

```bash
brew upgrade dotctl         # if installed via homebrew
dotctl update               # self-update from GitHub release
dotctl update --from-source # rebuild from source
```

## Development

```bash
make build    # build binary
make test     # run tests
make install  # build + copy to ~/.local/bin/
make release  # cross-compile darwin/arm64 + darwin/amd64
make clean    # remove artifacts
```

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT
