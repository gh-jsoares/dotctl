# Shell Integration

dotctl provides shell integration for zsh and bash. It's required for context switching to work (the binary can't modify the parent shell's environment).

## Setup

Choose one:

### Option A: eval at startup (~5ms)

```zsh
# In .zshrc
eval "$(dotctl shell-init zsh)"
```

### Option B: pre-generated file (faster)

```bash
# Generate once (and after config changes)
dotctl shell-init install

# In .zshrc
source ~/.local/share/dotctl/init.zsh
```

Option B avoids spawning a subprocess at shell startup.

## What it provides

### `ctx` function

Wraps `dotctl ctx` and sources the env file afterward:

```bash
ctx work       # switch to work
ctx personal   # switch to personal
ctx            # show current context
```

This is a shell function (not an alias) because it needs to `source` the env file in the current shell. The binary handles symlinks and writes the file; the function sources it.

### Chdir hook

On every directory change, checks for a `.dotctx` file in the current directory or ancestors (up to 10 levels). If found and the preferred context doesn't match the active one:

```
⚠ This repo prefers context 'work'. Current: 'personal'. Run: ctx work
```

The hook is lightweight — it's a `stat` check, not a subprocess call.

### Guarded commands

From `[[guards]]` in your config. Each entry generates a shell function that wraps the real command with a confirmation prompt when run outside the required context.

```toml
# In config.toml
[[guards]]
command = "awscreds"
context = "work"
message = "awscreds writes to AWS/kube/docker config for the active context."

[[guards]]
command = "terraform"
context = "work"
```

When you run `awscreds` while in `personal` context:

```
⚠ awscreds writes to AWS/kube/docker config for the active context.
Current context: personal. Continue? [y/N]
```

If `message` is omitted, a default is generated: "Running COMMAND outside 'CONTEXT' context."

The guard doesn't block — it warns and asks for confirmation. This handles edge cases where you legitimately need to run a "work" tool in personal context.

### Env sourcing

Shell startup sources the env file (`~/.local/state/dotctl/env`) automatically. This file contains:
- `DOTCTL_CONTEXT` — the active context name
- All `[env]` vars from the active context
- All resolved `[lazy]` secrets (cached from last switch)

The file is just a series of `export` statements — sourcing it is a file read, zero cost.

## Bash support

```bash
# In .bashrc
eval "$(dotctl shell-init bash)"
```

The bash variant provides the same `ctx` function. Chdir hook and guards work identically.

Note: `dotctl shell-init install` currently only writes the zsh file. For bash, use the eval approach.
