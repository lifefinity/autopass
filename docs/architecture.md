# Architecture

## Overview

autopass is a CLI tool that wraps interactive commands in a pseudo-terminal, watches their output for configurable patterns, and automatically responds with encrypted secrets. It functions like Linux `expect` but with built-in secret management.

## High-Level Flow

```
┌─────────────────────────────────────────────────────────┐
│                      User                                │
│   autopass myserver --then "ls" --then "exit"            │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│                   cmd/ (CLI Layer)                        │
│                                                          │
│  root.go: Dispatch profile name → runProfileWithSteps()  │
│  run.go:  Decrypt secret, build engine.Options           │
│  version.go: Print version/commit/build info             │
│  helpers.go: Load data, derive encryption key            │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│              internal/engine/ (Execution Engine)          │
│                                                          │
│  engine.go:       Run() entry point                      │
│  pty_windows.go:  ConPTY process + I/O loop              │
│  pty_unix.go:     PTY process + I/O loop                 │
│  matcher.go:      Regex pattern matching                 │
│  stepper.go:      Post-login command sequencing          │
└─────────────────────┬───────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────┐
│              Pseudo-Terminal (ConPTY / PTY)               │
│                                                          │
│  Child process (kinit, ssh, psql, etc.)                 │
└─────────────────────────────────────────────────────────┘
```

## Component Details

### CLI Layer (`cmd/`)

Responsible for:
- Parsing commands and flags (via Cobra)
- Loading profile data from disk
- Deriving encryption key from SSH private key
- Decrypting stored secrets
- Building `engine.Options` and calling `engine.Run()`

Key dispatch logic:
- `autopass <name>` → `root.go` → `runProfileWithSteps()` (loads profile, decrypts, runs)
- `autopass add/update/list/remove/version` → respective command handlers

### Engine (`internal/engine/`)

The core execution engine. Platform-agnostic interface with platform-specific PTY implementations.

#### engine.go

- `Run(opts Options)` — entry point
- `stripAnsi(s string)` — removes ANSI escape sequences before matching
- Delegates to `runWithPTY()` (platform-specific)

#### matcher.go

- Compiles regex patterns at startup
- `Check(line string) *MatchResult` — tests a line against all patterns
- Returns the response string and hidden flag on match

#### stepper.go

- Manages post-login command sequences (`--then`, `--script`)
- Activated after the first pattern match (password sent)
- Watches for shell prompt pattern, sends next command in sequence
- Thread-safe (mutex-protected)

#### pty_windows.go / pty_unix.go

Platform-specific PTY management:

**Windows (ConPTY):**
```
CreatePipe (input) → pipeIn (we write) / ptyIn (PTY reads)
CreatePipe (output) → pipeOut (we read) / ptyOut (PTY writes)
CreatePseudoConsole(size, ptyIn, ptyOut)
CreateProcess with PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE
```

**I/O Loop (both platforms):**
```
Goroutine 1: stdin → pipeWriter (forward user input)
Goroutine 2: pipeReader → stdout + pattern matching
  - Accumulates output in line buffer
  - On newline: process complete line
  - On 100ms timeout: process partial line (handles prompts without newline)
  - Strips ANSI codes before matching
  - On match: writes response to pipeWriter
```

### Crypto (`internal/crypto/`)

```
SSH Private Key File
       │
       ├─ Parse with ssh.ParseRawPrivateKey()
       │   (supports ed25519, RSA, ECDSA; passphrase-protected keys)
       │
       ▼
   Raw key bytes
       │
       ├─ HKDF-SHA256
       │   Salt: "autopass-salt-v1"
       │   Info: "autopass-v1"
       │
       ▼
   256-bit AES key (derived, never stored)
       │
       ├─ Encrypt: AES-256-GCM (random 12-byte nonce)
       │   Output: nonce || ciphertext || tag
       │
       └─ Decrypt: AES-256-GCM
           Input: nonce || ciphertext || tag
           Output: plaintext secret
```

### Data (`internal/data/`)

Single JSON file at `~/.autopass/data.json` (permissions 0600):

```json
{
  "ssh_key": "~/.ssh/id_ed25519",
  "profiles": {
    "prod": {
      "command": "ssh deploy@prod-server",
      "patterns": [{"match": "(?i)password:", "hidden": true}],
      "secret": "base64(nonce+ciphertext+tag)",
      "timeout": "30s"
    },
    "mydb": {
      "command": "psql -h db.example.com -U admin mydb",
      "patterns": [{"match": "(?i)password", "hidden": true}],
      "secret": "base64(nonce+ciphertext+tag)",
      "prompt": "=>\\s*$",
      "timeout": "30s"
    }
  }
}
```

Responsibilities:
- Load/Save with JSON marshaling
- Validate regex patterns on load
- Prevent reserved names (add, update, list, remove, version, init, help)
- CRUD operations on profiles

## Data Flow: Pattern Matching

```
PTY Output (raw bytes)
       │
       ▼
   Accumulate in line buffer
       │
       ├─ Newline found? → Extract complete line
       │
       └─ 100ms timeout? → Flush partial line
              │
              ▼
       Strip ANSI escape codes
              │
              ▼
       Matcher.Check(cleanLine)
              │
              ├─ Match found → Write response + "\r\n" to PTY input
              │                 Activate Stepper
              │
              └─ No match → Stepper.Check(cleanLine)
                                    │
                                    ├─ Prompt matched + steps remaining
                                    │   → Write next step + "\r\n"
                                    │
                                    └─ No match → do nothing
```

## Security Model

1. **No plaintext secrets on disk** — all secrets encrypted with AES-256-GCM
2. **Key derived from SSH key** — no separate master password; leverages existing SSH key management
3. **Key never stored** — derived on-the-fly each invocation via HKDF
4. **Per-secret nonce** — each encryption uses a unique random nonce
5. **File permissions** — data.json created with 0600 (owner read/write only)
6. **Hidden input** — secrets read via `term.ReadPassword()` (no echo)
7. **Hidden response** — when `hidden: true`, the response is not echoed to terminal output

## Concurrency Model

The engine uses goroutines for non-blocking I/O:

```
Main goroutine:
  └─ WaitForSingleObject (process exit)
  └─ ClosePseudoConsole (triggers reader EOF)
  └─ wg.Wait()

Goroutine 1 (input forwarder):
  └─ io.Copy(pipeWriter, stdin)

Goroutine 2 (output reader + matcher):
  └─ Async read loop with select
  └─ Timer-based flush for partial lines
  └─ Pattern matching + response writing
```

The Stepper is mutex-protected for thread safety since it can be accessed from the output reader goroutine.
