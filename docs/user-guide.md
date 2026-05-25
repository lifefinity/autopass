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

This creates `~/.autopass/data.json` and configures your SSH key path for encryption.

## Adding a Profile

### Interactive Mode

```bash
autopass add myserver
```

Prompts for:
1. Command to run
2. Pattern to match
3. Secret (hidden input)

### With Flags

```bash
autopass add -c "ssh deploy@prod-server" -m "(?i)password:" prod
autopass add -c "psql -h db.example.com -U admin mydb" -m "(?i)password" -p "=>\s*$" mydb
autopass add -c "mysql -h db.example.com -u root -p" -m "(?i)password:" mysql-prod
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--command` | `-c` | Command to execute |
| `--match` | `-m` | Regex pattern to match in output |
| `--prompt` | `-p` | Shell prompt regex for post-login steps |
| `--timeout` | `-t` | Timeout per step (default: `30s`) |

## Running a Profile

```bash
autopass prod
```

This:
1. Loads the profile from `~/.autopass/data.json`
2. Derives the encryption key from your SSH key
3. Decrypts the stored secret
4. Launches the command in a pseudo-terminal
5. Watches output for the configured pattern
6. Auto-types the secret when matched

## Post-Login Commands

After the password prompt is answered, you can chain commands:

```bash
# Inline
autopass mydb --then "SELECT now();" --then "\q"

# From file
autopass mydb --script queries.sql

# Combined
autopass mydb --then "\timing on" --script queries.sql --then "\q"

# Override prompt pattern
autopass mydb --prompt "=>\s*$" --then "SELECT 1;"
```

### Script File Format

One command per line. Empty lines and `#` comments are ignored:

```sql
# Enable timing
\timing on

# Run query
SELECT * FROM users LIMIT 10;

# Exit
\q
```

## Managing Profiles

```bash
# List all profiles
autopass list

# Remove a profile
autopass remove prod

# Update only the secret
autopass update prod --secret

# Update the command
autopass update prod -c "ssh newuser@prod-server"

# Update match pattern and timeout
autopass update mysql-prod -m "(?i)enter password:" -t 60s

# Update multiple fields at once
autopass update mydb -c "psql -h newhost -U admin mydb" --secret
```

### Update Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--command` | `-c` | Update the command |
| `--match` | `-m` | Update the prompt pattern (regex) |
| `--prompt` | `-p` | Update the shell prompt pattern |
| `--timeout` | `-t` | Update the timeout |
| `--secret` | | Prompt for a new secret |

Only specified flags are changed; unspecified fields remain unchanged.

## Version

```bash
autopass version
autopass --version
```

## Pattern Tips

| Pattern | Matches |
|---------|---------|
| `(?i)password:` | Case-insensitive "password:" |
| `(?i)password for` | Kerberos-style prompt |
| `=>\s*$` | PostgreSQL prompt |
| `\$\s*$` | Bash/shell prompt |
| `#\s*$` | Root prompt |
| `mysql>` | MySQL prompt |

## Troubleshooting

### "Profile not found"

Make sure the profile exists — check with `autopass list`.

### Secret not being sent

1. Check your pattern matches the actual output with `autopass list`
2. The prompt text from the program may contain ANSI escape codes — autopass strips these before matching
3. Try a broader regex (e.g. `(?i)pin` instead of exact match)

### SSH key passphrase

If your SSH key has a passphrase, autopass will prompt for it once per invocation to derive the encryption key.

## Platform Support

| Platform | Terminal Method |
|----------|---------------|
| Windows 10+ | ConPTY (Windows Pseudo Console API) |
| Linux | PTY (creack/pty) |
| macOS | PTY (creack/pty) |
