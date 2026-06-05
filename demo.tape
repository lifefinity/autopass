# VHS demo tape - https://github.com/charmbracelet/vhs
# Run: vhs demo.tape

Output demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"

Type "# autopass - encrypted expect in one binary"
Enter
Sleep 1s

Type "autopass add mwinit -c 'mwinit -s' -m 'PIN:'"
Enter
Sleep 500ms

Type "Enter your secret PIN: "
Sleep 300ms
Type "****"
Enter
Sleep 1s

Type "# Now run it anytime - secret is encrypted at rest"
Enter
Sleep 800ms

Type "autopass mwinit"
Enter
Sleep 2s

Type "# List all profiles"
Enter
Sleep 500ms

Type "autopass list"
Enter
Sleep 2s

Type "# Export (secrets excluded) / Import"
Enter
Sleep 500ms

Type "autopass export profiles.json"
Enter
Sleep 1s

Type "# Dry-run to preview"
Enter
Sleep 500ms

Type "autopass mwinit --dry-run"
Enter
Sleep 2s
