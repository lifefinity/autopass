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

If you try to add a name that already exists, autopass will error and suggest `update` instead.

---

## SSH

```bash
# Add
autopass add -c "ssh deploy@prod-server" -m "password:" prod
autopass add -c "ssh -p 2222 admin@bastion.example.com" -m "password:" bastion
autopass add -c "ssh -J bastion db-internal" -m "password:" db-jump

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

# Run
autopass mydb           # connects to PostgreSQL, auto-fills password
autopass mysql-prod     # connects to MySQL
autopass oracle-prod    # connects to Oracle
```

### Database with Post-Login Commands

```bash
# PostgreSQL: enable timing, set schema, run queries
autopass add -c "psql -h db.example.com -U admin mydb" -m "password" \
  -p "=>\s*$" --then "\timing on" --then "SET search_path TO app;" mydb

# MySQL: run a script file after login
autopass mydb --script queries.sql

# Combined: post-login setup + script + exit
autopass mydb --then "\timing on" --script queries.sql --then "\q"
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

### Post-Exit Commands

Run commands after the main process exits (e.g., verify auth succeeded):

```bash
autopass add -c "kinit admin@CORP.COM" -m "Password:" \
  --after "klist" \
  --after "echo 'Kerberos ticket refreshed'" \
  krb
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
