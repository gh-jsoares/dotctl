# Commands

## dotctl ctx

Switch developer context.

```
dotctl ctx [name]         # switch to context
dotctl ctx current        # show active context
dotctl ctx default NAME   # set default for new shells
dotctl ctx list           # list available contexts
```

Without arguments, shows the current context.

## dotctl bootstrap

Bootstrap a fresh machine. See [Bootstrap](bootstrap.md) for full details.

```
dotctl bootstrap [flags]
```

| Flag | Description |
|------|-------------|
| `--dotfiles-remote` | Git remote for dotfiles repo |
| `--dotctl-remote` | Git remote for dotctl repo |
| `--dotfiles-path` | Local path for dotfiles repo |
| `--dotctl-path` | Local path for dotctl repo |
| `--default-context` | Default context to set (default: `personal`) |

## dotctl sync

Converge the system to the desired state. Run after pulling dotfiles changes.

```
dotctl sync [flags]
```

| Flag | Description |
|------|-------------|
| `--no-pull` | Skip `git pull` before syncing |
| `--dotfiles-only` | Only pull and stow (skip nix, sheldon, mise) |

Core steps (in order):
1. `git pull` — pull dotfiles repo
2. `submodule update` — sync and update submodules
3. `nix-darwin switch` — apply nix-darwin config
4. `commit flake.lock` — auto-commit if dirty
5. `stow -R` — re-stow all dotfile packages
6. `sheldon lock` — update zsh plugin lockfile
7. `mise install` — ensure all runtimes are present

After core steps, [plugins](plugins.md) run in dependency order.

Each step is skipped gracefully if not applicable (no flake, no stow dir, no mise).

## dotctl plugins

Manage plugins. See [Plugins](plugins.md) for full documentation.

```
dotctl plugins list              # show discovered plugins and status
dotctl plugins validate          # check all manifests for errors
dotctl plugins run <name>        # manually run a plugin's sync hook
```

## dotctl doctor

Validate environment health.

```
dotctl doctor
```

Built-in checks:
- State directory exists
- A context is set
- Context matches current working directory
- Env file exists
- Symlinks are valid (exist and point to real targets)
- Required tools are installed (nix, darwin-rebuild, stow, mise, tmux, git, op)

After built-in checks, any plugins with a `doctor` hook are run.

## dotctl project

Detect project context from the current directory.

```
dotctl project
```

Walks up the directory tree (up to 10 levels) looking for a `.dotctx` file. Reports the preferred context and whether it matches the active one.

Example `.dotctx` file:
```
context = "work"
```

## dotctl secrets

Manage secrets in 1Password. All subcommands accept `--account` which can be a 1Password account URL (e.g., `my.1password.com`) or a context name (e.g., `work`, `personal`) that resolves to the context's `op_account`.

Secrets are stored in the `dotctl` vault by default.

```
dotctl secrets get <reference>               # retrieve a single secret
dotctl secrets set <reference> <value>       # store a secret
dotctl secrets list [vault]                  # list items (defaults to "dotctl" vault)
dotctl secrets env                           # resolve all [lazy] refs for current context
```

| Flag | Applies to | Description |
|------|-----------|-------------|
| `--account` | all | 1Password account or context name |
| `--category` | set | Item category: "SSH Key", "API Credentials", "Password", "Secure Note", etc. (default: "Secure Note") |

### Examples

```bash
# Get a secret from the personal account
dotctl secrets get op://dotctl/github-token/credential --account personal

# Store an API key as API Credentials
dotctl secrets set op://dotctl/openai/api_key "sk-..." --account personal --category "API Credentials"

# List all items in the dotctl vault for work
dotctl secrets list --account work

# Refresh lazy secrets for current context
eval "$(dotctl secrets env)"
```

Note: lazy secrets are automatically resolved and cached during `ctx switch`. The `secrets env` command is for manual refresh only (e.g., after a secret rotates).

## dotctl shell-init

Output or install shell integration code.

```
dotctl shell-init zsh       # print zsh integration to stdout
dotctl shell-init bash      # print bash integration to stdout
dotctl shell-init install   # write to ~/.local/share/dotctl/init.zsh
```

## dotctl status

Show current environment state at a glance.

```
dotctl status
```

Displays:
- Active context (with mismatch warning if CWD doesn't match)
- Dotfiles git state (branch, ahead/behind, dirty files)
- Which sync steps would run
- Which plugins are active vs skipped

## dotctl update

Self-update the binary.

```
dotctl update                         # download latest release from GitHub
dotctl update --check                 # only check if an update is available
dotctl update --from-source           # git pull + go build (from current dir)
dotctl update --from-source --source-path /opt/dotctl
```

| Flag | Description |
|------|-------------|
| `--check` | Only check for updates, don't install |
| `--from-source` | Rebuild from source instead of downloading |
| `--source-path` | Path to dotctl source repo (with `--from-source`) |

Release-based update requires `dotctl.remote` in config (owner/repo are parsed from the URL). Source-based update defaults to the current directory.

A daily update check runs automatically at the end of `dotctl sync` — if a newer version is available, a notice is printed.

## dotctl completion

Generate shell completion scripts.

```
dotctl completion bash    # output bash completions
dotctl completion zsh     # output zsh completions
dotctl completion fish    # output fish completions
```

See [Shell Completions](completions.md) for static generation and caching strategies.

## dotctl version

Print version and commit hash.

```
dotctl version
# dotctl v0.3.2 (abc1234)
```
