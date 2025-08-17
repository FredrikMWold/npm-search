Output demo.gif

# Terminal appearance and environment
Set FontSize 24
Set FontFamily "MesloLGS NF"
Set Theme "Catppuccin Mocha"
Set Width 1800
Set Height 900
Set Padding 10
Set Margin 30
Set MarginFill "#674EFF"
Set BorderRadius 12
Set Shell zsh

# Build the app and prep a friendly alias without showing the noise
Hide
Type "go build -o npm-search ./cmd/npm-search && alias npm-search='./npm-search' && clear"
Enter
Show

# Launch the TUI
Type "npm-search"
Sleep 1200ms
Enter
Sleep 2500ms

# Search for a small package
Type "left-pad"
Sleep 1s
Enter
Sleep 3s

# Browse results
Down
Sleep 700ms
Down
Sleep 700ms
Down
Sleep 700ms
Down
Sleep 700ms
Up
Sleep 500ms

# Open details (sidebar highlights metadata)
Enter
Sleep 1500ms

# Install the selected package
Type "i"
Sleep 5s

# After install completes, the row shows "✔ Installed"
Sleep 2s

# Toggle help to showcase keybindings
Type "?"
Sleep 1500ms
Type "?"
Sleep 1200ms

# Also install the 3rd item using capital I (dev install)
# Move to the top to normalize position
Up
Sleep 250ms
Up
Sleep 250ms
Up
Sleep 250ms
Up
Sleep 250ms
Up
Sleep 400ms

# Go to the 3rd item (index 2)
Down
Sleep 600ms
Down
Sleep 600ms

# Install as dev
Type "I"
Sleep 7s

# Wait for the Installed badge
Sleep 1500ms

# Linger on the UI for a moment
Sleep 4s

# Exit the app
Ctrl+c
Sleep 1200ms

# Done
Sleep 600ms
