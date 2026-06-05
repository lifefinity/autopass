# autopass

[中文文档](docs/README_zh.md)

A CLI tool that automatically answers interactive prompts (passwords, PINs, passphrases) with encrypted secrets. Think `expect`, but simpler and with built-in secret management.

## Why autopass?

| Tool | Scripting | Secret Storage | Cross-Platform | Learning Curve |
|------|-----------|----------------|----------------|----------------|
| **autopass** | No script needed — one-liner setup | AES-256-GCM, key derived from SSH key | Windows (ConPTY) + Linux/macOS (PTY) | Minimal |
| expect/pexpect | TCL/Python scripts | None (plaintext in scripts) | Linux/macOS only | Moderate |
| sshpass | Single command only | Plaintext flag or env var | Linux only | Low |
| ansible vault | Playbook-level | Encrypted vault | Via Ansible | High |

## Installation

### Download Binary

Download the latest release from [Releases](https://github.com/lifefinity/autopass/releases/latest):

```bash
# Linux (amd64)
curl -sL https://github.com/lifefinity/autopass/releases/latest/download/autopass-linux-amd64 -o autopass
chmod +x autopass && sudo mv autopass /usr/local/bin/

# macOS (Apple Silicon)
curl -sL https://github.com/lifefinity/autopass/releases/latest/download/autopass-darwin-arm64 -o autopass
chmod +x autopass && sudo mv autopass /usr/local/bin/

# Windows — download autopass-windows-amd64.exe from Releases page
```

### Build from Source

```bash
git clone https://github.com/lifefinity/autopass.git
cd autopass && make build
```

## Quick Start

```bash
# Build
make build    # → bin/autopass.exe (with version info)

# 1. Add a profile
autopass add -c "ssh user@server" -m "password:" myserver

# 2. Run it — password auto-filled
autopass myserver

# 3. Check what you have
autopass list
```

## How It Works

```
autopass myserver
    │
    ├─ Load profile from ~/.autopass/data.json
    ├─ Derive AES key from SSH private key (HKDF-SHA256)
    ├─ Decrypt stored secret
    ├─ Launch command in pseudo-terminal
    ├─ Watch output → match regex → auto-type secret
    └─ Process exits normally
```

## Examples

### Common Profiles

```bash
# SSH server
autopass add -c "ssh deploy@prod-server" -m "password:" prod

# PostgreSQL
autopass add -c "psql -h db.example.com -U admin mydb" -m "password" -p "=>\s*$" mydb

# MySQL
autopass add -c "mysql -h db.example.com -u root -p" -m "password:" mysql-prod

# Docker registry
autopass add -c "docker login registry.example.com -u ci" -m "password:" docker-reg

# Sudo
autopass add -c "sudo apt upgrade -y" -m "password" apt-upgrade

# Kerberos
autopass add -c "kinit admin@EXAMPLE.COM" -m "password for" krb

# Redis CLI (AUTH)
autopass add -c "redis-cli -h cache.example.com" -m "password:" redis

# FTP
autopass add -c "ftp files.example.com" -m "password:" ftp-files
```

### Post-Login Commands

Chain commands after the password is auto-filled:

```bash
# Run SQL queries after connecting
autopass mydb --then "SELECT now();" --then "\q"

# Execute a script file
autopass mydb --script queries.sql

# Combined
autopass mydb --then "\timing on" --script queries.sql --then "\q"
```

### Update a Profile

```bash
# Change only the secret
autopass update prod --secret

# Change the command
autopass update prod -c "ssh newuser@prod-server"

# Change match pattern and timeout
autopass update mysql-prod -m "enter password:" -t 60s

# Change description
autopass update prod -d "Production deployment server"

# Set post-login steps
autopass update mydb --then "\timing on" --then "SET search_path TO app;"

# Set post-exit commands
autopass update mwinit --after "date" --after "echo done"

# Enable case-sensitive matching
autopass update myserver --case-sensitive
```

### Quiet Mode

Run without terminal output (useful for scripts/CI):

```bash
# Silent execution — password still auto-filled
autopass mydb --quiet --script queries.sql

# Short form
autopass mydb -q --then "SELECT 1;"

# Capture output to file (without --quiet, stdout has PTY output)
autopass mydb --script queries.sql > result.txt
```

## Commands

| Command | Description |
|---------|-------------|
| `autopass <profile>` | Run a profile with auto-answering |
| `autopass add <profile>` | Create a new profile |
| `autopass update <profile>` | Update specific fields of a profile |
| `autopass list` | Show all profiles |
| `autopass remove <profile>` | Delete a profile |
| `autopass change-key <path>` | Switch to a new SSH key for encryption |
| `autopass export <file>` | Export profiles to JSON (without secrets) |
| `autopass import <file>` | Import profiles from JSON |
| `autopass backup <dir>` | Backup key + data to a directory |
| `autopass restore <dir>` | Restore key + data from a backup |
| `autopass completion <shell>` | Generate shell completion (bash/zsh/fish/powershell) |
| `autopass version` | Print version info |
| `autopass init` | First-time setup |

## Pattern Matching

- **Case-insensitive by default** — `"password:"` matches `Password:`, `PASSWORD:`, etc.
- Use `--case-sensitive` flag when adding/updating a profile for exact case matching
- Patterns are Go regular expressions (e.g. `"password|passphrase"` matches either)

## Backup & Restore

```bash
# Backup key and encrypted data to a directory
autopass backup /mnt/usb/autopass-backup

# Restore on a new machine
autopass restore /mnt/usb/autopass-backup

# Overwrite existing data
autopass restore ~/backup --force
```

Export/import profiles *without* the encryption key (for sharing configs):

```bash
autopass export profiles.json    # secrets excluded
autopass import profiles.json    # merge with existing (--force to overwrite)
```

## Security

- Secrets encrypted with **AES-256-GCM** (random nonce per secret)
- Encryption key derived from your **SSH private key** via HKDF-SHA256 — never stored on disk
- If no SSH key exists, a dedicated ed25519 key is auto-generated at `~/.autopass/autopass_key`
- Data file at `~/.autopass/data.json` with 0600 permissions
- No plaintext secrets anywhere

## Platform Support

| Platform | Method |
|----------|--------|
| Windows 10+ | ConPTY |
| Linux | PTY (creack/pty) |
| macOS | PTY (creack/pty) |

## Documentation

- [User Guide](docs/user-guide.md) — detailed usage, flags, troubleshooting
- [Architecture](docs/architecture.md) — component design, data flow, security model
- [Development](docs/development.md) — building, testing, contributing
