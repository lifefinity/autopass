# User Guide

## Installation

```bash
cd projects/autopass
go build -o autopass.exe .
```

Or use the Makefile:

```bash
make build        # Output: bin/autopass.exe
make install      # Install to GOPATH/bin
```

## First-Time Setup

On first use, autopass will auto-initialize. You can also run manually:

```bash
autopass init
```

This creates `~/.autopass/data.json` and configures encryption.

### How `init` works

1. If `--key <path>` is specified, uses that SSH key
2. Otherwise looks for `~/.ssh/id_ed25519`, `id_rsa`, or `id_ecdsa`
3. If no SSH key is found, generates a dedicated key at `~/.autopass/autopass_key`

If the selected key is passphrase-protected, you will be prompted once to verify access.

### Init Flags

| Flag | Description |
|------|-------------|
| `--key <path>` | Use an existing SSH private key instead of auto-detecting |
| `--no-passphrase` | Skip passphrase prompt for generated key |

### Examples

```bash
autopass init                              # Auto-detect or generate
autopass init --key ~/.ssh/id_ed25519      # Use a specific key
autopass init --no-passphrase              # Generate key without passphrase
```

## Adding a Profile

### Interactive Mode

```bash
autopass add myserver
```

Prompts for command, pattern, and secret.

### With Flags

```bash
autopass add -c "ssh deploy@prod-server" -m "password:" -d "Production server" prod
autopass add -c "psql -h db.example.com -U admin mydb" -m "password" -p "=>\s*$" -d "Main database" mydb
autopass add -c "mwinit -s -o" -m "PIN:" -d "Midway refresh" --after "date" mwinit
```

### Add Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--command` | `-c` | Command to execute |
| `--desc` | `-d` | Short description (shown in `autopass list`) |
| `--match` | `-m` | Regex pattern to match (can specify multiple) |
| `--prompt` | `-p` | Shell prompt regex for post-login `--then` steps |
| `--timeout` | `-t` | Timeout per match (default: `30s`) |
| `--case-sensitive` | | Match with exact case (default: case-insensitive) |
| `--then` | | Command to send inside session after login (can specify multiple) |
| `--after` | | Command to run in new shell after profile exits (can specify multiple) |

### --then vs --after

| | `--then` | `--after` |
|---|----------|-----------|
| **When** | Inside the running session | After process exits (exit code 0) |
| **Requires** | `-p` prompt pattern | Nothing |
| **Use for** | psql, mysql, ssh shell | mwinit, kinit, one-shot commands |
| **Runs in** | The PTY session | A new `sh -c` shell |

### Example: PostgreSQL with built-in steps

```bash
autopass add -c "psql -h localhost -U admin mydb" \
  -m "password" -p "=>\s*$" \
  --then "\timing on" --then "SET search_path TO public;" \
  -d "Local PG with timing" mydb
```

### Example: mwinit with post-exit command

```bash
autopass add -c "mwinit -s -o" -m "PIN:" \
  --after "date" --after "echo 'Midway refreshed'" \
  -d "Midway auth" mwinit
```

## Running a Profile

```bash
autopass <profile>
autopass <profile> --then "cmd"       # Additional session command
autopass <profile> --script file      # Commands from file
autopass <profile> --after "cmd"      # Run after process exits
autopass <profile> -e KEY=VALUE       # Inject environment variable
autopass <profile> --quiet            # Suppress terminal output
```

### Environment Variables

The child process inherits the current shell environment. Use `--env`/`-e` to inject additional variables:

```bash
autopass deploy -e HOST=prod.example.com -e PORT=5432
```

If the profile command uses shell variables (e.g., `ssh $USER@$HOST`), they are expanded at runtime from the inherited + injected environment.

## Post-Login Commands

After the password prompt is answered, chain commands inside the session:

```bash
autopass mydb --then "SELECT now();" --then "\q"
autopass mydb --script queries.sql
autopass mydb --then "\timing on" --script queries.sql --then "\q"
```

Profile-stored `--then` steps run first, then runtime `--then`/`--script` commands.

### Script File Format

One command per line. Empty lines and `#` comments are ignored:

```sql
# Enable timing
\timing on
SELECT * FROM users LIMIT 10;
\q
```

## Managing Profiles

```bash
autopass list                                    # List all profiles
autopass remove prod                             # Delete a profile
autopass update prod --secret                    # Change secret
autopass update prod -c "ssh newuser@host"       # Change command
autopass update prod -d "New description"        # Change description
autopass update prod --then "ls" --then "pwd"    # Set post-login steps
autopass update prod --after "notify-send done"  # Set post-exit commands
```

### Update Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--command` | `-c` | Update the command |
| `--desc` | `-d` | Update the description |
| `--match` | `-m` | Update the prompt pattern |
| `--prompt` | `-p` | Update the shell prompt pattern |
| `--timeout` | `-t` | Update the timeout |
| `--secret` | | Prompt for a new secret |
| `--case-sensitive` | | Enable exact-case matching |
| `--then` | | Replace post-login steps |
| `--after` | | Replace post-exit commands |

Only specified flags are changed; unspecified fields remain unchanged.

## Changing the Encryption Key

Switch all secrets to a new SSH key:

```bash
autopass change-key ~/.ssh/id_ed25519_new
```

This:
1. Decrypts all secrets with the current key (prompts for passphrase if needed)
2. Re-encrypts all secrets with the new key (prompts for passphrase if needed)
3. Updates the key reference in `data.json`

Both old and new keys can be passphrase-protected.

## Backup & Restore

### Backup

```bash
autopass backup /mnt/usb/autopass-backup
```

### Restore

```bash
autopass restore /mnt/usb/autopass-backup
autopass restore ~/backup --force           # Overwrite existing
```

## Export & Import

```bash
autopass export profiles.json              # Secrets excluded
autopass import profiles.json              # Merge with existing
autopass import profiles.json --force      # Overwrite on conflict
```

> Imported profiles have no secrets. Use `autopass update <name> --secret` to set them.

## Shell Completion

```bash
autopass completion bash    # eval "$(autopass completion bash)"
autopass completion zsh     # eval "$(autopass completion zsh)"
autopass completion fish    # autopass completion fish > ~/.config/fish/completions/autopass.fish
```

Tab-completion works for profile names: `autopass <TAB>`, `autopass update <TAB>`, `autopass remove <TAB>`.

## Pattern Tips

Patterns are **case-insensitive by default**. A partial match is enough.

| Pattern | Matches |
|---------|---------|
| `password` | "Password for user demo1:", "Enter password:", "PASSWORD:" |
| `pin` | "Midway PIN:", "Enter PIN:" |
| `Password for user .+:` | Any username in PostgreSQL prompts |
| `=>\s*$` | PostgreSQL shell prompt (for `-p` flag) |
| `\$\s*$` | Bash prompt |
| `mysql>` | MySQL prompt |

Use `--case-sensitive` only when you need exact matching.

## Troubleshooting

### "Profile not found"

Check with `autopass list`.

### Secret not being sent

1. Pattern may not match — try a broader regex (e.g. `password`)
2. ANSI escape codes are stripped before matching
3. Check timeout — increase with `autopass update <name> -t 60s`

### Process hangs after error

If the child process exits with an error (non-zero), autopass exits immediately. If it seems stuck, the child process may still be running — check with `ps`.

### SSH key passphrase

If your SSH key has a passphrase, autopass will prompt for it once per invocation.

## Platform Support

| Platform | Terminal Method |
|----------|---------------|
| Windows 10+ | ConPTY |
| Linux | PTY (creack/pty) |
| macOS | PTY (creack/pty) |
