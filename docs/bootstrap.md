# Bootstrap

`dotctl bootstrap` takes a fresh macOS machine from nothing to a fully configured developer environment. It's idempotent — safe to re-run at any point.

## Prerequisites

Only one thing is needed: the dotctl binary. Get it with:

```bash
curl -sSf https://raw.githubusercontent.com/OWNER/dotctl/main/install.sh | bash
```

## Running

```bash
dotctl bootstrap
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--dotfiles-remote` | Git remote for dotfiles repo | (interactive prompt) |
| `--dotfiles-path` | Local path for dotfiles repo | `~/.dotfiles` |
| `--default-context` | Context to set as default | `personal` |

If a config file exists (`~/.config/dotctl/config.toml`), values are read from it. Missing values are prompted interactively.

## Steps

Bootstrap runs these steps in order, skipping any that are already complete:

### 1. Xcode CLI Tools

Runs `xcode-select --install`. Skipped if already installed.

### 2. Nix

Installs Nix using the [Determinate Systems installer](https://determinate.systems/nix-installer/). Skipped if `nix` is already in PATH.

### 3. Pre-clone SSH Setup

Generates an SSH key and configures it for cloning your repos from GitHub. Interactive — asks for:
- **Key label** (e.g., `personal`) — used as the key filename suffix
- **Host alias** (e.g., `personal.github.com`) — written to `~/.ssh/config`

The flow:
1. Generates `~/.ssh/id_ed25519_<label>`
2. Writes an SSH host block to `~/.ssh/config`
3. Tests the connection with `ssh -T`
4. If not authorized: displays the public key and waits for you to add it to GitHub

Skipped if repos are already cloned.

### 4. Clone Dotfiles

Clones your dotfiles repository to the configured path.

### 5. nix-darwin Switch

Runs your nix-darwin flake to install packages, configure system settings, and set up Homebrew casks.

On first run (no `darwin-rebuild` in PATH yet), uses:
```bash
nix run nix-darwin -- switch --flake <path>#<hostname>
```

Subsequent runs use `darwin-rebuild switch --flake ...` directly.

Skipped if no `flake.nix` exists in the dotfiles path.

### 6. Post-clone SSH Setup

Reads all context TOML files from `<dotfiles>/contexts/` and sets up SSH keys for each context that defines an `[ssh]` section.

For each context:
1. If `key_source` is set and `op` is available: retrieves the private key from 1Password
2. Otherwise: generates a new key and prompts you to add it to GitHub
3. Writes host blocks to `~/.ssh/config`

Handles deduplication — if two contexts claim the same SSH host with the same key, only one setup runs. If they claim the same host with *different* keys, bootstrap errors with a clear message.

### 7. Create Context Directories

Creates the directories referenced in context symlink targets (e.g., `~/.aws-work`, `~/.aws-personal`, `~/.docker-work`).

### 8. Stow Dotfiles

Runs `stow -S` for all directories under `<dotfiles>/stow/`. Skipped if no `stow/` directory exists.

### 9. mise install

Runs `mise install` to install all configured runtimes. Skipped if `mise` isn't available (it gets installed by nix-darwin in step 5).

### 10. Set Default Context + Doctor

Sets the default context and runs `dotctl doctor` to validate everything is healthy.

## After Bootstrap

```bash
# Start a new shell (or source the integration)
eval "$(dotctl shell-init zsh)"

# Switch to your preferred context
ctx personal
```

## Re-running

Every step checks its preconditions:
- Xcode installed? Skip.
- Nix in PATH? Skip.
- Repos already cloned? Skip pre-clone SSH and clone steps.
- No flake.nix? Skip nix-darwin.
- No stow dir? Skip stow.

This means you can interrupt bootstrap at any point and resume later without side effects.
