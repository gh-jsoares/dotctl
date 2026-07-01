# dotctl

A developer environment orchestrator for macOS. Manages context switching between work and personal environments, bootstraps fresh machines, and keeps your system in sync.

## What it does

dotctl solves the problem of maintaining isolated developer environments on a single machine. If you work on personal projects and company projects from the same laptop, you've likely dealt with:

- AWS credentials from work leaking into personal projects (or vice versa)
- Kubernetes contexts from multiple clusters polluting a single kubeconfig
- Docker auth tokens from work ECR registries conflicting with Docker Hub
- Git commits going out with the wrong identity
- Nexus/Artifactory registries being set when you're working on open source

dotctl provides **context switching** that isolates these concerns at the filesystem level, plus orchestration to bootstrap and maintain your full environment.

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

## Installation

### From release (fresh machine)

```bash
curl -sSf https://raw.githubusercontent.com/OWNER/dotctl/main/install.sh | bash
```

The install script detects your OS/arch, downloads the binary, and places it in `/usr/local/bin`.

### From source

```bash
git clone git@github.com:OWNER/dotctl.git
cd dotctl
make install  # builds and copies to ~/.local/bin/
```

## Configuration

dotctl reads its configuration from `~/.config/dotctl/config.toml` (or `$XDG_CONFIG_HOME/dotctl/config.toml`).

```toml
machine = "personal-mbp"

[dotfiles]
path = "~/dotfilesv2/dotfiles"
remote = "git@personal.github.com:youruser/dotfiles.git"

[dotctl]
path = "~/dotfilesv2/dotctl"
remote = "git@personal.github.com:youruser/dotctl.git"
repo_owner = "youruser"
repo_name = "dotctl"

[ssh.hosts]
personal = "personal.github.com"
work = "work.github.com"

[[guards]]
command = "awscreds"
context = "work"
message = "This will write to your current context's AWS/kube/docker config."
```

The dotfiles path is resolved in order: `$DOTFILES_DIR` env var → config file → `~/.dotfiles` default.

## Context Switching

### The problem

Many tools write to hardcoded paths:
- `~/.aws/credentials` — AWS CLI, awscreds, etc.
- `~/.kube/config` — kubectl, awscreds --kube, helm
- `~/.docker/config.json` — docker login, awscreds --ecr

If a work tool writes 15 AWS profiles into `~/.aws/credentials`, those bleed into your personal shell. There's no way to tell `awscreds` to write elsewhere — it hardcodes the path.

### The solution

dotctl uses **directory-level symlinks** for tools that hardcode paths, and **environment variables** for tools that respect them:

```
~/.aws  → symlink → ~/.aws-work OR ~/.aws-personal
~/.kube → symlink → ~/.kube-work OR ~/.kube-personal
$DOCKER_CONFIG   → ~/.docker-work OR ~/.docker-personal
```

When you run `ctx work`, the symlinks flip and the env vars change. Work tools write into the work directory. Personal tools write into the personal directory. They never cross.

### Context definitions

Contexts are defined in your dotfiles repo as TOML files:

```
dotfiles/contexts/work.toml
dotfiles/contexts/personal.toml
```

Example `work.toml`:

```toml
[identity]
git_config = "config-work"
ssh_key = "id_ed25519_work"

[symlinks]
"~/.aws" = "~/.aws-work"
"~/.kube" = "~/.kube-work"

[env]
DOCKER_CONFIG = "~/.docker-work"
NPM_CONFIG_REGISTRY = "https://nexus.company.com/repository/npm/"

[env.lazy]
ARTIFACTORY_TOKEN = "op://Work/Artifactory/token"
```

Fields:
- `[identity]` — git config variant and SSH key name
- `[symlinks]` — directory symlinks to create (source → target)
- `[env]` — environment variables to export
- `[env.lazy]` — secrets resolved on-demand via `dotctl secrets env` (never at shell startup)

### Switching

```bash
ctx work         # switch current shell to work context
ctx personal     # switch to personal
ctx              # show current context (alias for dotctl ctx current)
ctx default work # set default for all new shells
```

`ctx` is a shell function (not the binary directly) because it needs to source environment variables into the current shell. The binary handles symlinks and writes the env file; the shell function sources it.

### What happens on switch

1. Symlinks are updated (`~/.aws` → `~/.aws-work`)
2. Git config symlink updates (`~/.config/git/config-current` → `config-work`)
3. Env file is written to `~/.local/state/dotctl/env`
4. Current context is recorded in `~/.local/state/dotctl/current-context`
5. If inside tmux: server environment is updated (new panes inherit the context)

## Commands

### `dotctl ctx`

Switch developer context (see above).

```
dotctl ctx [name]       # switch to context
dotctl ctx current      # show active context
dotctl ctx default NAME # set default for new shells
```

### `dotctl bootstrap`

Bootstrap a fresh machine from scratch. Idempotent — safe to re-run.

```
dotctl bootstrap
```

Steps performed:
1. Install Xcode CLI tools (if missing)
2. Install Nix via Determinate Systems installer (if missing)
3. Install 1Password CLI (if missing)
4. Write SSH config with host aliases (from `[ssh.hosts]` config)
5. Clone dotfiles and dotctl repos
6. Create context directories (`~/.aws-work`, `~/.aws-personal`, etc.)
7. Run `darwin-rebuild switch` (if `flake.nix` exists)
8. Run `stow -S` for all packages (if `stow/` directory exists)
9. Run `mise install` (if mise is available)

Each step checks whether it's already complete and skips if so.

### `dotctl sync`

Converge the system to the desired state. Run after pulling dotfiles changes.

```
dotctl sync
```

Steps:
1. `darwin-rebuild switch` — apply nix-darwin config
2. `stow -R` — re-stow all dotfile packages
3. `mise install` — ensure all runtimes are present

Steps that aren't applicable (no flake, no stow dir, no mise) are skipped gracefully.

### `dotctl doctor`

Validate environment health.

```
dotctl doctor
```

Checks:
- State directory exists
- A context is set
- Env file exists
- Symlinks are valid (exist and point to real targets)
- Required tools are installed (nix, stow, mise, tmux, git, op)

### `dotctl project`

Detect project context from the current directory.

```
dotctl project
```

Looks for a `.dotctx` file in the current directory or ancestors (up to 10 levels). Reports the preferred context and whether it matches the active one.

Example `.dotctx` file in a repo root:
```
context = "work"
```

### `dotctl secrets`

Retrieve secrets from 1Password.

```
dotctl secrets get "op://Vault/Item/field"   # print a secret value
dotctl secrets env                            # resolve all [env.lazy] refs for current context
```

`secrets env` outputs `export` statements — pipe to `eval` or source from a file. This is intentionally NOT run at shell startup (it's slow). Run it manually when you need the lazy secrets.

### `dotctl shell-init`

Output or install shell integration code.

```
dotctl shell-init zsh           # print to stdout (for eval)
dotctl shell-init bash          # bash variant
dotctl shell-init install       # write to ~/.local/share/dotctl/init.zsh
```

The shell integration provides:
- `ctx` function (wrapper that sources env after switching)
- `chpwd` hook (warns if repo prefers a different context)
- Guarded command wrappers (from `[[guards]]` config)

### `dotctl update`

Self-update.

```
dotctl update               # download latest release from GitHub
dotctl update --from-source # git pull + go build in the dotctl repo
```

### `dotctl version`

Print version and commit.

```
dotctl version
# dotctl v0.1.0 (abc1234)
```

## Shell Integration

Add to your `.zshrc`:

```zsh
# Option A: eval at shell startup (~5ms)
eval "$(dotctl shell-init zsh)"

# Option B: source a pre-generated file (faster, 0 subprocess cost)
source ~/.local/share/dotctl/init.zsh
```

For option B, run `dotctl shell-init install` once (and after config changes).

### What it provides

**`ctx` function** — wraps `dotctl ctx` and sources the env file afterward:
```bash
ctx work      # switch to work
ctx personal  # switch to personal
ctx           # show current
```

**chdir hook** — on every `cd`, checks for `.dotctx` in the directory tree. If found and the preferred context doesn't match the active one, prints a warning:
```
⚠ This repo prefers context 'work'. Current: 'personal'. Run: ctx work
```

**Guarded commands** — from `[[guards]]` in config. Generates shell functions that wrap the real command with a confirmation prompt when run outside the required context:
```
⚠ This will write to your current context's AWS/kube/docker config.
Current context: personal. Continue? [y/N]
```

## Guarded Commands

Configure in `~/.config/dotctl/config.toml`:

```toml
[[guards]]
command = "awscreds"
context = "work"
message = "awscreds writes to AWS/kube/docker config for the active context."

[[guards]]
command = "terraform"
context = "work"

[[guards]]
command = "helm"
context = "work"
message = "helm is using the kubeconfig for your current context."
```

Each entry generates a shell function wrapper. If `message` is omitted, a default is generated: "Running COMMAND outside 'CONTEXT' context."

The guard doesn't block — it warns and asks for confirmation. This handles edge cases where you legitimately need to run a "work" tool in personal context.

## Project Context Detection

Place a `.dotctx` file in any repository root:

```
context = "work"
```

When you `cd` into that repo (or any subdirectory), the shell hook checks if your active context matches. If not:

```
⚠ This repo prefers context 'work'. Current: 'personal'. Run: ctx work
```

You can also check manually:
```bash
dotctl project
# Project dotctx: /path/to/repo/.dotctx
# Preferred context: work
# Current context: personal
#
# ⚠ Context mismatch. Run: ctx work
```

This is intentionally a **warning only** — no automatic switching. You stay in control.

## File Layout

### State (managed by dotctl at runtime)

```
~/.local/state/dotctl/
├── env               # sourceable env vars for current context
├── current-context   # name of active context ("work" or "personal")
└── default-context   # name of default context for new shells
```

### Context directories (created by bootstrap)

```
~/.aws-work/          # awscreds writes here when context=work
~/.aws-personal/      # personal AWS config
~/.kube-work/         # EKS clusters, work kubectl contexts
~/.kube-personal/     # personal k8s contexts
~/.docker-work/       # ECR auth, work Docker config
~/.docker-personal/   # Docker Hub, personal registries
```

### Symlinks (managed by ctx switch)

```
~/.aws              → ~/.aws-work OR ~/.aws-personal
~/.kube             → ~/.kube-work OR ~/.kube-personal
~/.config/git/config-current → config-work OR config-personal
```

## Design Decisions

### Why directory symlinks instead of file symlinks?

Tools like `awscreds` may write multiple files into `~/.aws/`. If we symlink individual files, a new file written by the tool lands in the wrong place. Symlinking the entire directory guarantees all writes are isolated — even for files we haven't anticipated.

### Why a shell function for `ctx`?

A subprocess can't modify its parent's environment. `dotctl ctx work` writes the env file and flips symlinks, but the `export` statements need to be sourced in the current shell. The thin `ctx()` function calls the binary then sources the result.

### Why not Home Manager?

Home Manager ties dotfile content to Nix evaluation. This creates coupling between "what packages are installed" and "what my config files say." Stow keeps these concerns separate — Nix handles packages, stow handles config file placement, and the config files themselves are plain text in the dotfiles repo.

### Why TOML for context definitions?

TOML is simple, readable, and has good Go support. Context files are small and rarely change. TOML's `[section]` syntax maps cleanly to the identity/symlinks/env/lazy structure.

### Why not auto-switch context?

Automatic context switching on `cd` is dangerous. Imagine a script that `cd`s into a work repo then runs a command — it would silently change your AWS credentials mid-execution. The warning-only approach keeps you informed without creating footguns.

## Development

```bash
make build    # build binary
make test     # run tests
make install  # build + copy to ~/.local/bin/
make release  # cross-compile darwin/arm64 + darwin/amd64
make clean    # remove artifacts
```

Version and commit are injected at build time via ldflags:
```bash
go build -ldflags "-X github.com/OWNER/dotctl/cmd.version=v1.0.0 -X github.com/OWNER/dotctl/cmd.commit=$(git rev-parse --short HEAD)" .
```

## License

MIT
