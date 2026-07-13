# Shell Completions

dotctl supports completions for bash, zsh, and fish via the `completion` command.

## Quick Setup

Add to your shell config:

```sh
# bash (~/.bashrc)
eval "$(dotctl completion bash)"

# zsh (~/.zshrc, before compinit)
eval "$(dotctl completion zsh)"

# fish (~/.config/fish/config.fish)
dotctl completion fish | source
```

## Static Generation (recommended)

Evaluating completions on every shell startup adds latency. For faster shells, generate a static file and source it instead.

### zsh

```sh
# Generate once (or regenerate after upgrading dotctl)
dotctl completion zsh > "${XDG_CACHE_HOME:-$HOME/.cache}/zsh/completions/_dotctl"
```

Make sure the completions directory is in your `fpath`:

```sh
fpath=("${XDG_CACHE_HOME:-$HOME/.cache}/zsh/completions" $fpath)
autoload -Uz compinit && compinit
```

To regenerate daily in the background:

```sh
_comp_dir="${XDG_CACHE_HOME:-$HOME/.cache}/zsh/completions"
if [[ ! -f "$_comp_dir/_dotctl" || ! $(find "$_comp_dir/_dotctl" -newermt "24 hours ago" -print) ]]; then
  dotctl completion zsh >| "$_comp_dir/_dotctl" 2>/dev/null &|
fi
```

### bash

```sh
# Generate once
dotctl completion bash > "${XDG_DATA_HOME:-$HOME/.local/share}/bash-completion/completions/dotctl"
```

bash-completion will source files from this directory automatically.

### fish

```sh
# Generate once
dotctl completion fish > "${XDG_CONFIG_HOME:-$HOME/.config}/fish/completions/dotctl.fish"
```

Fish automatically loads files from `~/.config/fish/completions/`.
