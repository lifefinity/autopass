# Examples

Complete cookbook of autopass profiles for common tools.

## How It Works

```bash
# 1. Add a profile (one-time setup)
autopass add -c "ssh deploy@prod" -m "password:" myserver

# 2. Run it — password auto-filled every time
autopass myserver

# 3. Manage
autopass list              # show all profiles
autopass update myserver   # modify fields
autopass remove myserver   # delete
autopass myserver --dry-run  # preview without running
```

## Profile Naming

Each profile has a **unique name** — it is the key you use to run it. Names:
- Must be unique across all profiles (regardless of tool type)
- Must start with a letter/number, contain only `a-z A-Z 0-9 . - _`
- Cannot conflict with subcommands (`add`, `list`, `remove`, etc.)
- The same name can be reused with different `--service` (`-s`) values for multi-service disambiguation (e.g. `prod` with `-s ssh` and `prod` with `-s pg`)

If you try to add a name that already exists (with the same service), autopass will error and suggest `update` instead.

If you try to add a name that already exists, autopass will error and suggest `update` instead.

---

## SSH

```bash
# Add
autopass add -c "ssh deploy@prod-server" -m "password:" prod
autopass add -c "ssh -p 2222 admin@bastion.example.com" -m "password:" bastion
autopass add -c "ssh -J bastion db-internal" -m "password:" db-jump

# SSH key with passphrase (auto-fills the key unlock prompt)
autopass add -c "ssh -i ~/.ssh/deploy_key user@prod" -m "passphrase" prod-key

# Multi-service: same name, different service types
autopass add -c "ssh deploy@prod-server" -m "password:" -s ssh prod
autopass add -c "psql -h prod-db -U admin" -m "password" -s pg prod
autopass prod -s ssh    # connects via SSH
autopass prod -s pg     # connects to PostgreSQL

# Run
autopass prod           # connects to prod-server, auto-fills password
autopass prod-key       # unlocks private key, then connects
# Run
autopass prod           # connects to prod-server, auto-fills password
autopass bastion        # connects to bastion on port 2222
```

## Databases

```bash
# Add
autopass add -c "psql -h db.example.com -U admin mydb" -m "password" -p "=>\s*$" mydb
autopass add -c "mysql -h db.example.com -u root -p" -m "password:" mysql-prod
autopass add -c "sqlplus admin@orcl" -m "password:" oracle-prod
autopass add -c "mongosh mongodb://admin@db.example.com:27017/mydb" -m "password:" mongo-prod
autopass add -c "redis-cli -h cache.example.com" -m "password:" redis

# Multi-service: disambiguate same host by database type
autopass add -c "psql -h shared-db -U app" -m "password" -s pg shared-db
autopass add -c "mysql -h shared-db -u app -p" -m "password:" -s mysql shared-db
autopass shared-db -s pg      # PostgreSQL
autopass shared-db -s mysql   # MySQL

# Run
autopass mydb           # connects to PostgreSQL, auto-fills password
autopass mysql-prod     # connects to MySQL
autopass oracle-prod    # connects to Oracle
```

### --then vs --after

Two different hooks for two different moments:

| Flag | When | Use for |
|------|------|---------|
| `--then` | **Inside** the session, after password is filled | Run SQL, set config, execute commands in the shell |
| `--after` | **After** the process exits | Verify result, notify, chain next tool |

```
┌─────────────────────────────────────────────────────────┐
│  autopass mydb --then "\timing on" --after "echo done"  │
│                                                         │
│  1. Launch: psql -h db -U admin                         │
│  2. Auto-fill password when prompted                    │
│  3. --then: type "\timing on" into psql session  ← inside│
│  4. User interacts with psql...                         │
│  5. User exits psql (\q or Ctrl-D)                      │
│  6. --after: run "echo done" in parent shell     ← after│
└─────────────────────────────────────────────────────────┘
```

### Database with --then (Post-Login Commands)

```bash
# Run SQL after login
autopass mydb --then "SELECT now();" --then "\q"

# Enable timing + set schema (saved in profile)
autopass add -c "psql -h db.example.com -U admin mydb" -m "password" \
  -p "=>\s*$" --then "\timing on" --then "SET search_path TO app;" mydb

# Run a script file
autopass mydb --script queries.sql

# Combined
autopass mydb --then "\timing on" --script queries.sql --then "\q"
```

### Kerberos with --after (Post-Exit Commands)

```bash
# Verify ticket was obtained after kinit exits
autopass add -c "kinit admin@CORP.COM" -m "Password:" \
  --after "klist" \
  --after "echo 'Kerberos ticket refreshed'" \
  krb

# SSH + notify when disconnected
autopass add -c "ssh deploy@prod" -m "password:" \
  --after "echo 'Disconnected from prod at $(date)'" \
  prod
```

## System Administration

```bash
# Sudo
autopass add -c "sudo apt upgrade -y" -m "password" apt-upgrade

# Switch user
autopass add -c "su - postgres" -m "password:" su-postgres

# Kerberos
autopass add -c "kinit admin@CORP.EXAMPLE.COM" -m "Password for" krb
```

## Containers & CI

```bash
# Docker registry login
autopass add -c "docker login registry.example.com -u ci" -m "password:" docker-reg

# Podman login
autopass add -c "podman login ghcr.io -u bot" -m "password:" ghcr

# Helm registry
autopass add -c "helm registry login registry.example.com -u ci" -m "password:" helm-reg
```

## File Transfer

```bash
# SFTP
autopass add -c "sftp deploy@files.example.com" -m "password:" sftp-prod

# SCP (when PasswordAuthentication is enabled)
autopass add -c "scp backup.tar.gz admin@backup-server:/backups/" -m "password:" scp-backup
```

## Git (HTTPS)

```bash
# Git push over HTTPS (credential prompt)
autopass add -c "git push origin main" -m "Password for" git-push

# Git clone private repo
autopass add -c "git clone https://github.com/org/private-repo.git" -m "Password for" git-clone
```

## TOTP / Two-Factor Authentication

```bash
# TOTP-only (VPN token, MFA prompt)
autopass add -c "vpn-connect" -m "Verification code:" vpn --totp

# Password + TOTP (two-step login)
autopass add -c "ssh admin@secure-server" -m "password:" \
  --totp-match "Verification code:" secure-ssh

# AWS CLI MFA (token code prompt)
autopass add -c "aws sts get-session-token" -m "Enter MFA code:" aws-mfa --totp

# Run — both password and TOTP auto-filled
autopass secure-ssh
```

## Advanced Patterns

### Multiple Prompts

A profile can match multiple patterns (all share the same secret):

```bash
autopass add -c "ssh user@server" \
  -m "password:" \
  -m "Password:" \
  myserver
```

### Dry-Run

Preview what autopass will do without executing:

```bash
autopass myserver --dry-run
# Output:
# Command: ssh user@server
# Timeout: 30s
# Patterns:
#   - (?i)password:
```

### Quiet Mode (for scripts)

```bash
# No PTY output — just auto-fill and exit
autopass mydb -q --script queries.sql > result.txt
```

### Timeout Control

```bash
# Long-running session (5 min timeout)
autopass add -c "psql -h slow-db.example.com -U admin" -m "password" -t 5m psql-slow
```

## Backup & Migration

```bash
# Export profiles (no secrets) for sharing
autopass export team-profiles.json

# Import on another machine (merge, skip conflicts)
autopass import team-profiles.json

# Import with overwrite
autopass import team-profiles.json --force

# Full backup (key + encrypted data)
autopass backup /mnt/usb/autopass-backup

# Restore on new machine
autopass restore /mnt/usb/autopass-backup
```
