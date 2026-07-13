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
2. Git configs are generated (identity per context, `includeIf` by project dir)
3. `current-context` state file is written
4. Env file is generated with all `[env]` vars + resolved `[lazy]` secrets
5. Context-dirs mapping is written (for shell mismatch warnings)
6. If inside tmux: server environment is updated (new panes inherit)
7. Shell function sources the env file in the current shell

## Idempotency

Switching to the context you're already in is a no-op — unless the state is inconsistent (e.g., symlinks are missing or the env file doesn't exist). In that case, it re-applies everything to repair the state.

## Git identity

dotctl generates git config files from context TOML definitions:

```
~/.config/git/
├── config              # generated: includes config-shared + includeIf rules
├── config-shared       # stowed: core, delta, merge settings
├── config-work         # generated: [user] name, email, signing key
└── config-personal     # generated: [user] name, email, signing key
```

Identity is determined by **project directory**, not active context:
- The default context identity applies everywhere
- `includeIf "gitdir:~/projects/work/**"` overrides with work identity

This means repos under `~/projects/work/` always use the work identity, even if your active context is personal. No accidental commits with the wrong email.

The identity fields come from your context TOML:

```toml
[identity]
name = "João Soares"
email = "jsoares.public@gmail.com"
ssh_key = "id_ed25519_personal"
gpg_key_source = "op://Personal/GPG/private_key"
```

## Default context

```bash
ctx default work    # new shells start in work context
```

The default is loaded by shell integration on startup (sources the env file).

## Mismatch warnings

dotctl warns when your CWD is in a project dir that doesn't match your active context:

```
⚠ wrong context: in work dir but ctx is personal
```

This fires on:
- `cd` into a mismatched project dir (shell `chpwd` hook)
- `dotctl ctx` switch while already in a mismatched dir
- `dotctl status` and `dotctl doctor`

No automatic switching — just a warning. You stay in control.

## Prompt integration

dotctl exports `DOTCTL_CONTEXT` and `DOTCTL_CONTEXT_ICON` to the env file. Configure your prompt (starship, p10k, etc.) to display the active context.

Example starship config:

```toml
[custom.dotctl_ctx]
command = 'echo "${DOTCTL_CONTEXT_ICON:-⚙} ${DOTCTL_CONTEXT}"'
when = '[ -n "$DOTCTL_CONTEXT" ]'
style = "bold purple"
format = "[$output]($style) "
```

Set the icon per context in your TOML:

```toml
[prompt]
icon = "🏠"
```

## Tmux integration

When you switch context inside tmux:
- The tmux server environment is updated
- New panes/windows inherit the new context
- Existing panes keep their env until you source again

Your tmux status bar can read `~/.local/state/dotctl/current-context` to display the active context.
