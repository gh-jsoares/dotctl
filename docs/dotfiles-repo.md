# Dotfiles Repo Structure

dotctl expects your dotfiles repo to follow a specific layout. The repo contains your nix-darwin config, stow packages, and context definitions.

## Required structure

```
dotfiles/
├── contexts/              # context definitions (required for ctx switching)
│   ├── work.toml
│   └── personal.toml
├── flake.nix              # nix-darwin entry point (optional)
├── flake.lock
└── stow/                  # GNU Stow packages (optional)
    ├── zsh/
    ├── tmux/
    ├── git/
    └── ...
```

Only `contexts/` is required. Everything else is optional — dotctl skips steps that don't apply.

## Contexts directory

Each TOML file in `contexts/` defines one context. The filename (without `.toml`) is the context name.

See [Configuration](configuration.md) for the full context TOML format.

## Stow packages

Each subdirectory under `stow/` is a [GNU Stow](https://www.gnu.org/software/stow/) package. The directory structure mirrors your home directory:

```
stow/
├── zsh/
│   └── .config/
│       └── zsh/
│           ├── .zshrc
│           └── conf.d/
│               └── aliases.zsh
├── git/
│   └── .config/
│       └── git/
│           ├── config
│           ├── config-work
│           └── config-personal
├── tmux/
│   └── .config/
│       └── tmux/
│           └── tmux.conf
└── nvim/
    └── .config/
        └── nvim/
            └── init.lua
```

`dotctl sync` runs `stow -R` which creates symlinks from your home directory into these package directories.

### Adding a new stow package

1. Create a directory under `stow/` matching the tool name
2. Mirror the target path structure (relative to `$HOME`)
3. Place your config files
4. Run `dotctl sync` (or `stow -S -d stow -t ~ <package>`)

## Nix-darwin flake

If a `flake.nix` exists at the root of your dotfiles repo, `dotctl bootstrap` and `dotctl sync` will run nix-darwin.

The flake output is resolved as `<dotfiles-path>#<hostname>`. Your flake should have a `darwinConfigurations.<hostname>` output.

Example `flake.nix`:
```nix
{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    nix-darwin.url = "github:LnL7/nix-darwin";
    nix-darwin.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, nix-darwin }: {
    darwinConfigurations."My-MacBook" = nix-darwin.lib.darwinSystem {
      modules = [ ./nix/darwin.nix ];
    };
  };
}
```

### Recommended nix layout

```
dotfiles/
├── flake.nix
├── flake.lock
└── nix/
    ├── darwin.nix         # system settings, services
    ├── packages.nix       # CLI packages
    ├── homebrew.nix       # casks, GUI apps (managed declaratively)
    └── hosts/
        ├── personal-mbp.nix
        └── work-mbp.nix
```

## Git config pattern

For context-aware git identity, structure your git stow package like:

```
stow/git/.config/git/
├── config              # base config
├── config-work         # work identity
└── config-personal     # personal identity
```

Base `config`:
```gitconfig
[include]
  path = config-current

[includeIf "gitdir:~/work/"]
  path = config-work

[includeIf "gitdir:~/personal/"]
  path = config-personal
```

`config-work`:
```gitconfig
[user]
  name = Your Name
  email = you@company.com
```

`config-personal`:
```gitconfig
[user]
  name = Your Name
  email = you@personal.com
```

dotctl creates the `config-current` symlink pointing to the active context's git config. The `includeIf` rules handle repos in standard paths automatically; the symlink is the fallback for repos elsewhere.

## SSH host aliases

Each context can define an SSH host alias in its `[ssh]` section:

```toml
[ssh]
host = "personal.github.com"
github_user = "youruser"
```

This results in an `~/.ssh/config` entry:
```
Host personal.github.com
  HostName github.com
  User git
  IdentityFile ~/.ssh/id_ed25519_personal
  IdentitiesOnly yes
```

Clone repos using the alias:
```bash
git clone git@personal.github.com:youruser/repo.git
```

This lets you use multiple GitHub identities on one machine without conflicts.

## Private dotfiles repo

dotctl works with private repos. The bootstrap flow handles SSH key setup before cloning, so your dotfiles repo can be private.

The install script (`install.sh`) downloads only the pre-built dotctl binary from a public GitHub release — it doesn't need access to your dotfiles repo.

Typical setup:
- `dotctl` repo: **public** (so `install.sh` works without auth)
- `dotfiles` repo: **private** (contains your configs, context definitions, secrets references)

If you want dotctl private too, distribute the binary another way (e.g., direct download link, internal artifact store) and skip the install script.

## Minimal example

The smallest useful dotfiles repo:

```
dotfiles/
└── contexts/
    └── personal.toml
```

```toml
# contexts/personal.toml
[identity]
git_config = "config-personal"

[env]
DOCKER_CONFIG = "~/.docker-personal"
```

This gives you context switching with no nix-darwin, no stow, no mise. Add those layers as needed.
