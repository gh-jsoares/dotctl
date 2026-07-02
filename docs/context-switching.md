# Context Switching

Context switching is dotctl's core feature. It isolates developer environments at the filesystem level so work and personal tools never interfere.

## Why filesystem isolation?

Many tools write to hardcoded paths:
- `~/.aws/credentials` — AWS CLI, awscreds, etc.
- `~/.kube/config` — kubectl, helm, awscreds --kube
- `~/.docker/config.json` — docker login, awscreds --ecr

You can't tell these tools to write elsewhere. If a work CLI writes 15 AWS profiles into `~/.aws/credentials`, those are visible in your personal shell.

## How it works

dotctl uses two isolation mechanisms:

### Directory-level symlinks

For tools that hardcode paths (like `~/.aws`):

```
~/.aws  → ~/.aws-work       (when context = work)
~/.aws  → ~/.aws-personal   (when context = personal)
```

The entire directory is symlinked, not individual files. This guarantees ALL files a tool writes stay isolated — even files you haven't anticipated.

### Environment variables

For tools that respect env vars (like Docker with `DOCKER_CONFIG`):

```bash
DOCKER_CONFIG=~/.docker-work       # when context = work
DOCKER_CONFIG=~/.docker-personal   # when context = personal
```

## Switch flow

When you run `ctx work`:

1. Symlinks are updated (`~/.aws` → `~/.aws-work`, etc.)
2. Git config symlink updates (`~/.config/git/config-current` → `config-work`)
3. `current-context` state file is written
4. Env file is generated with all `[env]` vars + resolved `[lazy]` secrets
5. If inside tmux: server environment is updated (new panes inherit)
6. Shell function sources the env file in the current shell

## Idempotency

Switching to the context you're already in is a no-op — unless the state is inconsistent (e.g., symlinks are missing or the env file doesn't exist). In that case, it re-applies everything to repair the state.

## Git identity

dotctl manages git identity via config file symlinks:

```
~/.config/git/
├── config              # base config with includeIf rules
├── config-work         # [user] name, email, signing key for work
├── config-personal     # [user] name, email, signing key for personal
└── config-current      # symlink → config-work OR config-personal
```

The base `config` uses both approaches:
- `includeIf "gitdir:~/work/"` — path-based auto-detection
- `include path = config-current` — fallback for repos outside standard paths

This means repos under `~/work/` always use the work identity regardless of context, while random clones in `/tmp` use whatever context is active.

## Default context

```bash
ctx default work    # new shells start in work context
```

The default is loaded by shell integration on startup (sources the env file).

## Project detection

Place a `.dotctx` file in any repo:
```
context = "work"
```

The shell chdir hook warns on mismatch:
```
⚠ This repo prefers context 'work'. Current: 'personal'. Run: ctx work
```

No automatic switching — just a warning. You stay in control.

## Tmux integration

When you switch context inside tmux:
- The tmux server environment is updated
- New panes/windows inherit the new context
- Existing panes keep their env until you source again

Your tmux status bar can read `~/.local/state/dotctl/current-context` to display the active context.
