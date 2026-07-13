# Changelog

## 0.5.3 — 2026-07-13

### Added

- auto-generate changelog with git-cliff on release

## 0.5.2 — 2026-07-13

### Added

- add prompt icon to context, export DOTCTL_CONTEXT_ICON

## 0.5.1 — 2026-07-13

### Added

- copy public keys to clipboard during bootstrap
- align bootstrap output with sync UI style
- wire GPG setup into bootstrap steps

## 0.5.0 — 2026-07-13

### Added

- context mismatch warnings + GPG bootstrap

### Fixed

- increase demo sleep times for readability

## 0.4.5 — 2026-07-13

### Fixed

- projects plugin arithmetic causing exit 1 under set -e

## 0.4.4 — 2026-07-13

### Fixed

- projects plugin shows context name in output

## 0.4.3 — 2026-07-13

### Fixed

- projects plugin creates dirs for all contexts

## 0.4.2 — 2026-07-13

### Added

- refresh context env on sync

## 0.4.1 — 2026-07-13

### Fixed

- projects plugin sources context env to get PROJECTS_DIR

## 0.4.0 — 2026-07-13

### Added

- builtin plugins with disable support
- add record-demo script to generate and push gif

### Documentation

- document builtin plugins, disabling, and PROJECTS_DIR
- add vhs demo tape, workflow (assets branch), and gif in README
- add status, completion, update --check commands and homebrew install

### Fixed

- record-demo uses tmp dir to avoid stash/checkout
- install stow in demo workflow

## 0.3.2 — 2026-07-13

### Added

- run mise prune after install to remove orphaned tools

### Fixed

- version comparison strips v prefix to avoid false update notices

## 0.3.1 — 2026-07-13

### Added

- improved update flow with version check and daily update notifications

## 0.3.0 — 2026-07-13

### Added

- add shell completions, status command, and plugin tests

## 0.2.3 — 2026-07-13

### Fixed

- interactive steps (sudo, stow conflicts) bypass spinner and pipe

## 0.2.2 — 2026-07-13

### Added

- improved sync output with dimmed subprocess lines, spinners, and progress counters

### Documentation

- add contributing guide, CoC, issue/PR templates, and badges

## 0.2.1 — 2026-07-13

### Added

- polished terminal UI with lipgloss + stow conflict resolution

### Documentation

- add dotctl wrapper snippets for ctx/sync auto-sourcing

## 0.2.0 — 2026-07-13

### Added

- dynamic plugin system for user-defined orchestration steps

### Documentation

- add vibe coded disclaimer
- add plugin system documentation and update references

## 0.1.36 — 2026-07-10

### Fixed

- remove quarantine attribute after grimoire install

## 0.1.35 — 2026-07-10

### Fixed

- create ~/.local/bin before grimoire install

## 0.1.34 — 2026-07-10

### Added

- add grimoire install step to sync

## 0.1.33 — 2026-07-10

### Fixed

- remove shell exec after sync, let shell wrapper handle reload

## 0.1.32 — 2026-07-10

### Fixed

- run submodule sync before update to pick up URL changes

## 0.1.31 — 2026-07-09

### Added

- add tmux reload step, make aerospace reload non-fatal

## 0.1.30 — 2026-07-09

### Fixed

- check LaunchAgents instead of LaunchDaemons for pm2 startup

## 0.1.29 — 2026-07-09

### Fixed

- use pm2 startOrRestart with ecosystem config for simple-bar-server

## 0.1.28 — 2026-07-09

### Added

- add simple-bar-server sync step (pm2 install, startup, auto-start)

## 0.1.27 — 2026-07-09

### Added

- add --dotfiles-only flag to sync (skips nix, sheldon, mise)

## 0.1.26 — 2026-07-09

### Added

- add aerospace reload to sync, show Übersicht setup reminder
- add submodule update step to sync

## 0.1.25 — 2026-07-07

### Added

- auto-commit and push flake.lock changes after nix-darwin switch

## 0.1.24 — 2026-07-07

### Added

- add git pull step to sync with --no-pull flag

## 0.1.23 — 2026-07-07

### Added

- add sheldon lock to sync, exec fresh shell after sync

## 0.1.22 — 2026-07-07

### Added

- exec fresh shell after bootstrap, add brew/nix to shell-init PATH

## 0.1.21 — 2026-07-06

### Fixed

- sudo keep-alive uses -n flag and 30s interval

## 0.1.20 — 2026-07-06

### Added

- auto-elevate to sudo when update lacks write permission

## 0.1.19 — 2026-07-06

### Added

- cache sudo credentials at start of bootstrap with keep-alive

## 0.1.18 — 2026-07-06

### Fixed

- add nix-darwin and homebrew paths to PATH after switch

## 0.1.17 — 2026-07-06

### Added

- auto-derive dotctl remote in config from dotfiles remote

## 0.1.16 — 2026-07-06

### Fixed

- write SSH config before verification so host aliases resolve

## 0.1.15 — 2026-07-06

### Fixed

- download brew installer to file to preserve TTY stdin

## 0.1.14 — 2026-07-06

### Fixed

- let Homebrew installer prompt for sudo interactively

## 0.1.13 — 2026-07-06

### Added

- add Homebrew install step before nix-darwin

## 0.1.12 — 2026-07-06

### Fixed

- use nix-darwin/nix-darwin (repo moved from LnL7)

## 0.1.11 — 2026-07-06

### Fixed

- use config machine name (default: "default") for flake ref

## 0.1.10 — 2026-07-06

### Fixed

- restore sudo for first-run nix-darwin switch

## 0.1.9 — 2026-07-06

### Fixed

- use detected hostname directly for nix-darwin flake ref

## 0.1.8 — 2026-07-06

### Fixed

- prompt for flake config name when hostname doesn't match

## 0.1.7 — 2026-07-06

### Fixed

- drop sudo from first-run nix-darwin, use full flake ref

## 0.1.6 — 2026-07-06

### Fixed

- resolve nix path for sudo and hostname mismatch in bootstrap

## 0.1.5 — 2026-07-06

### Fixed

- run nix-darwin switch with sudo for system activation

## 0.1.4 — 2026-07-06

### Fixed

- source nix into current process after install

## 0.1.3 — 2026-07-06

### Fixed

- prompt for repo name instead of full remote URL

## 0.1.2 — 2026-07-06

### Fixed

- only write config after dotfiles clone succeeds

## 0.1.1 — 2026-07-02

### Fixed

- wait for Xcode CLI tools download before starting Nix
- remove hardcoded paths from examples

## 0.1.0 — 2026-07-02

### Added

- two-phase SSH bootstrap, lazy secret caching, bug fixes
- initial dotctl implementation

### Documentation

- rewrite documentation, simplify config

### Fixed

- update README owner, make test script portable
- bootstrap interactive prompts, ctx list, doctor darwin-rebuild

