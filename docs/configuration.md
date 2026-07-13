# Configuration

dotctl uses two configuration layers: a global config file for tool settings, and context TOML files in your dotfiles repo for per-context definitions.

## Config File

Location: `~/.config/dotctl/config.toml` (or `$XDG_CONFIG_HOME/dotctl/config.toml`)

```toml
machine = "personal-mbp"

[dotfiles]
path = "~/dotfiles"
remote = "git@personal.github.com:youruser/dotfiles.git"

[dotctl]
remote = "git@personal.github.com:youruser/dotctl.git"

[[guards]]
command = "awscreds"
context = "work"
message = "awscreds writes to AWS/kube/docker config for the active context."
```

### Fields

| Section | Field | Description |
|---------|-------|-------------|
| `dotfiles` | `path` | Local path to your dotfiles repo |
| `dotfiles` | `remote` | Git remote URL for cloning |
| `dotctl` | `remote` | Git remote URL (used for self-update release lookup) |
| `machine` | | Machine identifier (for nix-darwin host-specific config) |
| `guards` | | Array of guarded command definitions (see [Shell Integration](shell-integration.md)) |
| `plugins` | `disabled` | List of plugin names to skip (builtin or user-defined) |

Example with disabled plugins:

```toml
[plugins]
disabled = ["projects"]
```

### Resolution order

The dotfiles path is resolved in this order:
1. `$DOTFILES_DIR` environment variable
2. `config.toml` → `[dotfiles].path`
3. `~/.dotfiles` (default fallback)

### Auto-generated on bootstrap

If no config file exists when you run `dotctl bootstrap`, one is created from your interactive answers and flags. You don't need to write it manually on a fresh machine.

## Context Definitions

Context files live in your dotfiles repo at `contexts/<name>.toml`.

### Example: `contexts/work.toml`

```toml
[ssh]
host = "work.github.com"
github_user = "youruser-work"
key_source = "op://Work/SSH Key/private key"

[identity]
git_config = "config-work"
ssh_key = "id_ed25519_work"

[symlinks]
"~/.aws" = "~/.aws-work"
"~/.kube" = "~/.kube-work"

[env]
DOCKER_CONFIG = "~/.docker-work"
NPM_CONFIG_REGISTRY = "https://nexus.company.com/repository/npm/"

[lazy]
ARTIFACTORY_TOKEN = "op://Work/Artifactory/token"
```

### Example: `contexts/personal.toml`

```toml
[ssh]
host = "personal.github.com"
github_user = "youruser"

[identity]
git_config = "config-personal"
ssh_key = "id_ed25519_personal"

[symlinks]
"~/.aws" = "~/.aws-personal"
"~/.kube" = "~/.kube-personal"

[env]
DOCKER_CONFIG = "~/.docker-personal"
```

### Sections

#### `[ssh]`

SSH configuration for this context's GitHub access.

| Field | Description |
|-------|-------------|
| `host` | SSH host alias (e.g., `personal.github.com`) |
| `github_user` | GitHub username for this identity |
| `key_source` | 1Password reference for retrieving the SSH key (optional) |

If `key_source` is set, bootstrap will retrieve the private key from 1Password. Otherwise it generates a new key and prompts you to add it to GitHub.

#### `[identity]`

| Field | Description |
|-------|-------------|
| `git_config` | Which git config variant to symlink (`config-work`, `config-personal`) |
| `ssh_key` | SSH key filename in `~/.ssh/` |

#### `[symlinks]`

Maps source paths to target directories. On context switch, each source becomes a symlink to the corresponding target.

```toml
[symlinks]
"~/.aws" = "~/.aws-work"      # ~/.aws → ~/.aws-work
"~/.kube" = "~/.kube-work"    # ~/.kube → ~/.kube-work
```

#### `[env]`

Static environment variables exported when this context is active. Written to the env file on switch.

```toml
[env]
DOCKER_CONFIG = "~/.docker-work"
```

#### `[lazy]`

Secrets resolved from 1Password on context switch and cached in the env file. Values must be `op://` references.

```toml
[lazy]
ARTIFACTORY_TOKEN = "op://Work/Artifactory/token"
```

These are resolved once (via `op read`) when you switch context. The resolved values are cached in the env file so subsequent shells get them without re-auth. If `op` isn't authenticated, the secret is skipped with a warning — the switch still succeeds.

To manually refresh (e.g., after a secret rotates):
```bash
dotctl secrets env | source  # or: eval "$(dotctl secrets env)"
```

## State Files

dotctl stores runtime state in `~/.local/state/dotctl/` (or `$XDG_STATE_HOME/dotctl/`):

```
~/.local/state/dotctl/
├── env               # sourceable env vars for current context (0600)
├── current-context   # name of active context
└── default-context   # name of default context for new shells
```

The env file has `0600` permissions since it may contain resolved secrets.
