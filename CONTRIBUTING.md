# Contributing to dotctl

Thanks for wanting to contribute! This guide covers the basics.

## Getting Started

```bash
git clone git@github.com:gh-jsoares/dotctl.git
cd dotctl
make build   # verify it compiles
make test    # run tests
```

Requires Go 1.26+ and macOS (darwin) for full functionality.

## Development Workflow

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Run `make build` and `make test`
4. Commit with [conventional commits](https://www.conventionalcommits.org/) (e.g. `feat:`, `fix:`, `docs:`)
5. Open a PR against `main`

## Project Structure

```
cmd/              CLI commands (cobra)
internal/
  config/         Configuration loading
  context/        Context switching logic
  orchestrator/   Core sync steps (git, nix, stow, etc.)
  plugin/         Plugin discovery, conditions, ordering, execution
  ui/             Terminal output (lipgloss styling)
docs/             User-facing documentation
```

## Writing Plugins

If you're adding personal orchestration steps, those belong as **plugins** in your dotfiles repo, not in dotctl itself. See [docs/plugins.md](docs/plugins.md) for the full guide.

Contributions to the plugin _system_ (new condition types, execution features, etc.) are welcome here.

## What Makes a Good PR

- Solves one thing — don't bundle unrelated changes
- Includes a clear description of what and why
- Passes CI (`go vet`, `go test`, `go build`)
- Follows existing code style (no linter configured — just match what's there)

## Reporting Bugs

Use the [bug report template](https://github.com/gh-jsoares/dotctl/issues/new?template=bug_report.yml) and include:
- What you expected vs what happened
- Your OS version and Go version
- Relevant config (sanitized)

## Feature Requests

Open an issue with the [feature request template](https://github.com/gh-jsoares/dotctl/issues/new?template=feature_request.yml). Describe the problem you're solving, not just the solution you want.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). Be kind.
