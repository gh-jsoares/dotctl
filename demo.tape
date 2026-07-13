Output demo.gif

Set Shell "zsh"
Set FontSize 14
Set FontFamily "JetBrainsMono Nerd Font"
Set Width 1200
Set Height 700
Set Theme "ayu"
Set TypingSpeed 75ms
Set Padding 20
Set Framerate 60
Set Margin 20
Set MarginFill "#1A1F29"
Set BorderRadius 10
Set WindowBar Colorful

Env TERM "xterm-256color"
Env COLORTERM "truecolor"
Env CLICOLOR_FORCE "1"

# Show status overview
Type "dotctl status"
Enter
Sleep 3s

# Run sync (dotfiles only, no pull)
Type "dotctl sync --dotfiles-only --no-pull"
Enter
Sleep 15s

# List plugins
Type "dotctl plugins list"
Enter
Sleep 3s

# Show help
Type "dotctl --help"
Enter
Sleep 4s
