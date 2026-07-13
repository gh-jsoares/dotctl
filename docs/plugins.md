# Plugins

Plugins let you extend dotctl's `sync`, `bootstrap`, and `doctor` commands with custom shell scripts. They live in your dotfiles repo — not in dotctl itself — so your orchestration is portable and version-controlled alongside your configs.

## Overview

A plugin is a directory under `<dotfiles>/.dotctl/plugins/` containing a `plugin.toml` manifest and one or more hook scripts:

```
.dotctl/plugins/
├── aerospace/
│   ├── plugin.toml
│   └── sync.sh
├── grimoire/
│   ├── plugin.toml
│   ├── sync.sh
│   └── doctor.sh
└── my-tool/
    ├── plugin.toml
    ├── sync.sh
    ├── bootstrap.sh
    └── doctor.sh
```

dotctl discovers plugins automatically. No registration step is needed — drop a directory with a valid manifest and it runs on next `dotctl sync`.

## plugin.toml Reference

```toml
# Required
name = "my-plugin"
description = "What this plugin does"

# Hook scripts (all optional, relative to plugin directory)
[hooks]
sync = "sync.sh"           # runs during `dotctl sync`
bootstrap = "bootstrap.sh" # runs during `dotctl bootstrap`
doctor = "doctor.sh"       # runs during `dotctl doctor`

# Conditions — ALL must pass for the plugin to run (all fields optional)
[conditions]
paths_exist = ["stow/my-tool"]       # relative to dotfiles root
binaries_exist = ["my-tool"]         # must be on PATH
binaries_absent = []                 # skip if any are found on PATH
contexts = []                        # restrict to specific contexts (empty = any)
check = ""                           # shell command; skip plugin if exit != 0

# Ordering (optional)
[ordering]
after = ["stow"]                     # run after these core steps or plugins
before = []                          # run before these core steps or plugins
priority = 50                        # tie-breaker within same dependency tier (lower = earlier)

# Execution options (all optional)
[options]
continue_on_error = false            # if true, failure doesn't abort the sync
sudo = false                         # run the script with sudo
workdir = ""                         # working directory (relative to dotfiles, or absolute)
timeout = 120                        # kill script after N seconds (0 = no timeout)
[options.env]                        # extra environment variables
MY_VAR = "value"
INSTALL_DIR = "~/.local/bin"         # ~ is expanded to $HOME
```

## Core Step IDs

Use these in `ordering.after` or `ordering.before` to position your plugin relative to built-in steps:

| ID | Step |
|----|------|
| `git-pull` | `git pull --ff-only` on dotfiles repo |
| `submodule-update` | `git submodule sync` + `update --init` |
| `nix-darwin` | `sudo darwin-rebuild switch --flake` |
| `commit-flake-lock` | auto-commit dirty `flake.lock` |
| `stow` | `stow -R` all packages |
| `sheldon` | `sheldon lock --update` |
| `mise` | `mise install` |

If no `ordering` is specified, plugins run after all core steps (equivalent to `after = ["mise"]`).

## Environment Variables

Every hook script receives these environment variables in addition to the parent shell's environment:

| Variable | Value |
|----------|-------|
| `DOTCTL_DOTFILES_PATH` | Absolute path to the dotfiles repo |
| `DOTCTL_PLUGIN_DIR` | Absolute path to this plugin's directory |
| `DOTCTL_CONTEXT` | Current context name (e.g., `personal`, `work`) |
| `DOTCTL_MACHINE` | Machine name from config |
| `DOTCTL_HOOK` | Which hook is running: `sync`, `bootstrap`, or `doctor` |

Plus any variables defined in `[options.env]`.

## Conditions

All conditions are AND-ed — every specified condition must pass for the plugin to run. Omitted conditions are ignored (always pass).

### paths_exist

Check that files or directories exist. Paths are resolved relative to the dotfiles root unless absolute or starting with `~/`.

```toml
[conditions]
paths_exist = ["stow/aerospace", "nix/darwin.nix"]
```

### binaries_exist

Check that executables are on PATH.

```toml
[conditions]
binaries_exist = ["aerospace", "npm"]
```

### binaries_absent

Skip the plugin if any of these are already installed. Useful for install-once plugins.

```toml
[conditions]
binaries_absent = ["grimoire"]
```

### contexts

Restrict to specific contexts. Empty list means "run in any context".

```toml
[conditions]
contexts = ["work"]
```

### check

Run an arbitrary shell command. Plugin is skipped if exit code is non-zero.

```toml
[conditions]
check = "test -n \"$TMUX\""
```

## Ordering

Plugins are ordered using a directed acyclic graph (DAG). Each plugin can declare dependencies on core steps or other plugins.

```toml
[ordering]
after = ["stow", "nix-darwin"]   # this plugin runs after stow AND nix-darwin complete
before = ["tmux"]                # this plugin runs before the "tmux" plugin
priority = 10                    # lower number = earlier within same tier
```

If two plugins have no dependency relationship, `priority` determines the order (default: 50).

Circular dependencies are detected and reported as an error before any plugins run.

## Doctor Hook Convention

Doctor scripts validate plugin health. The contract:

- Exit `0` → healthy (stdout is ignored)
- Exit non-zero → unhealthy (stderr/stdout is shown as the failure message)

```bash
#!/usr/bin/env bash
if ! command -v grimoire &>/dev/null; then
  echo "grimoire not installed"
  exit 1
fi
```

## Examples

### Reload a window manager

```toml
name = "aerospace"
description = "Reload AeroSpace window manager config"

[hooks]
sync = "sync.sh"

[conditions]
binaries_exist = ["aerospace"]

[ordering]
after = ["stow"]

[options]
continue_on_error = true
```

```bash
#!/usr/bin/env bash
aerospace reload-config
```

### Install a binary from GitHub

```toml
name = "grimoire"
description = "Install grimoire binary from GitHub releases"

[hooks]
sync = "sync.sh"
doctor = "doctor.sh"

[conditions]
paths_exist = ["stow/grimoire"]

[ordering]
after = ["stow"]
priority = 10

[options]
timeout = 60
```

```bash
#!/usr/bin/env bash
set -euo pipefail
install_dir="$HOME/.local/bin"
mkdir -p "$install_dir"
curl -sL https://raw.githubusercontent.com/gh-jsoares/grimoire/main/install.sh | \
  GRIMOIRE_INSTALL_DIR="$install_dir" sh
xattr -c "$install_dir/grimoire" 2>/dev/null || true
```

### Start a managed service

```toml
name = "simple-bar-server"
description = "Start/restart simple-bar pm2 server"

[hooks]
sync = "sync.sh"

[conditions]
paths_exist = ["ubersicht/simple-bar-server"]
binaries_exist = ["npm"]

[ordering]
after = ["stow"]

[options]
timeout = 60
```

### Context-specific plugin

```toml
name = "work-vpn"
description = "Connect to work VPN after context switch"

[hooks]
sync = "sync.sh"

[conditions]
contexts = ["work"]
binaries_exist = ["warp-cli"]

[options]
continue_on_error = true
```

### Plugin that needs sudo

```toml
name = "dns-config"
description = "Write /etc/resolver entries for internal domains"

[hooks]
sync = "sync.sh"

[conditions]
contexts = ["work"]

[options]
sudo = true
```

## Builtin Plugins

dotctl ships with a set of builtin plugins embedded in the binary. These run alongside your user plugins and provide common functionality out of the box.

| Plugin | Description | Condition |
|--------|-------------|-----------|
| `projects` | Creates `PROJECTS_DIR` directory if set in context env | `PROJECTS_DIR` must be non-empty |

Builtins show as `(builtin)` in `dotctl plugins list`.

### Overriding a builtin

Create a plugin with the same name in your dotfiles. User plugins always take precedence over builtins:

```
.dotctl/plugins/projects/    ← your version wins
```

### Disabling plugins

Add to your `~/.config/dotctl/config.toml`:

```toml
[plugins]
disabled = ["projects"]
```

This works for any plugin — builtin or user-defined.

## CLI Commands

```bash
dotctl plugins list       # show discovered plugins and their enabled/disabled status
dotctl plugins validate   # check all manifests for errors (missing scripts, bad TOML)
dotctl plugins run <name> # manually run a single plugin's sync hook (skips conditions)
```

## Tips

- Keep plugin scripts idempotent — `dotctl sync` may run many times
- Use `continue_on_error = true` for non-critical plugins (config reloads, optional tools)
- Use `conditions.check` for runtime state that can't be expressed with path/binary checks
- The `DOTCTL_PLUGIN_DIR` variable lets scripts reference sibling files without hardcoding paths
- Plugin names share a namespace with core step IDs — don't name a plugin `stow` or `mise`
