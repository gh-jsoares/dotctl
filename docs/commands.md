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
dotctl sync
```

Steps:
1. `darwin-rebuild switch` — apply nix-darwin config
2. `stow -R` — re-stow all dotfile packages
3. `mise install` — ensure all runtimes are present

Each step is skipped gracefully if not applicable (no flake, no stow dir, no mise).

## dotctl doctor

Validate environment health.

```
dotctl doctor
```

Checks:
- State directory exists
- A context is set
- Env file exists
- Symlinks are valid (exist and point to real targets)
- Required tools are installed (nix, darwin-rebuild, stow, mise, tmux, git, op)

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

Retrieve secrets from 1Password.

```
dotctl secrets get "op://Vault/Item/field"   # retrieve a single secret
dotctl secrets env                            # resolve all [lazy] refs for current context
```

`secrets env` outputs `export` statements. Use it to manually refresh cached secrets:
```bash
eval "$(dotctl secrets env)"
```

Note: lazy secrets are automatically resolved and cached during `ctx switch`. This command is for manual refresh only (e.g., after a secret rotates).

## dotctl shell-init

Output or install shell integration code.

```
dotctl shell-init zsh       # print zsh integration to stdout
dotctl shell-init bash      # print bash integration to stdout
dotctl shell-init install   # write to ~/.local/share/dotctl/init.zsh
```

## dotctl update

Self-update the binary.

```
dotctl update                         # download latest release from GitHub
dotctl update --from-source           # git pull + go build (from current dir)
dotctl update --from-source --source-path ~/dotfilesv2/dotctl
```

Release-based update requires `dotctl.remote` in config (owner/repo are parsed from the URL). Source-based update defaults to the current directory.

## dotctl version

Print version and commit hash.

```
dotctl version
# dotctl v0.1.0 (abc1234)
```
