Output demo.gif

Set Shell "zsh"
Set FontSize 14
Env TERM "xterm-256color"
Env COLORTERM "truecolor"
Set Width 900
Set Height 600
Set Theme "Catppuccin Mocha"

# Show status
Type "dotctl status"
Enter
Sleep 3s

# Run a quick sync
Type "dotctl sync --dotfiles-only --no-pull"
Enter
Sleep 8s

# Show plugins
Type "dotctl plugins list"
Enter
Sleep 2s
