# QBridge

**QBridge** is a CLI control layer that sits between AI agents and your databases.  
Instead of giving agents direct DB access, you register credentials and define **profiles** — named policies that control which databases can be accessed and which SQL operations are permitted.

```
AI Agent  →  qbridge query --profile readonly "SELECT ..."  →  QBridge  →  Database
```

## Features

- 🔐 **Encrypted credential storage** — AES-256-GCM with Argon2id key derivation (`~/.qbridge/`)
- 🗄️ **Multi-database support** — PostgreSQL, MySQL, SQLite
- 👤 **Profiles** — bind databases to operation allowlists (`SELECT`, `INSERT`, `UPDATE`, `DELETE`, `DDL`)
- 🤖 **AI-agent friendly** — JSON output, `QBRIDGE_PASSWORD` env var for non-interactive use
- 🛡️ **Policy enforcement** — SQL intent is parsed (not string-matched) to prevent bypass

---

## Installation

### Build from source (requires Go 1.21+)

```bash
git clone https://github.com/jacksonfernando/qbridge
cd qbridge
go build -o qbridge .
sudo mv qbridge /usr/local/bin/
```

---

## Quick Start

### 1. Initialize

```bash
qbridge init
# Choose a master password: ****
# ✓ QBridge initialized.
```

### 2. Register a database

```bash
qbridge db add prod-postgres
# Database type (postgres/mysql/sqlite): postgres
# Host: db.example.com
# Port [5432]:
# Username: myuser
# Password: ****
# Database name: myapp
# SSL mode [prefer]:
# ✓ Database "prod-postgres" registered.
```

### 3. Create a profile

```bash
qbridge profile add readonly
# Available databases: prod-postgres
# Databases to include (comma-separated): prod-postgres
# Allowed operations — choices: SELECT, INSERT, UPDATE, DELETE, DDL
# Allow (comma-separated): SELECT
# ✓ Profile "readonly" created.
```

### 4. Query through the profile

```bash
qbridge query --profile readonly "SELECT id, name FROM users LIMIT 5"
```

Output:
```json
{
  "profile": "readonly",
  "database": "prod-postgres",
  "columns": ["id", "name"],
  "rows": [
    [1, "Alice"],
    [2, "Bob"]
  ],
  "rows_affected": 0
}
```

If the agent tries something outside the policy:
```bash
qbridge query --profile readonly "DELETE FROM users"
# Error: operation "DELETE" is not allowed by profile "readonly" (allowed: SELECT)
```

---

## Command Reference

### `qbridge init`
First-time setup. Creates `~/.qbridge/` and sets the master password.

---

### `qbridge db`

| Command | Description |
|---|---|
| `qbridge db add <name>` | Register a new database credential |
| `qbridge db list` | List all registered databases |
| `qbridge db test <name>` | Test connectivity to a database |
| `qbridge db remove <name>` | Remove a database credential |

---

### `qbridge profile`

| Command | Description |
|---|---|
| `qbridge profile add <name>` | Create a new access profile |
| `qbridge profile list` | List all profiles |
| `qbridge profile show <name>` | Show profile details |
| `qbridge profile edit <name>` | Edit profile databases or permissions |
| `qbridge profile remove <name>` | Delete a profile |

---

### `qbridge query`

```
qbridge query --profile <profile> [--db <database>] "<SQL>"
```

| Flag | Description |
|---|---|
| `--profile`, `-p` | Profile to use **(required)** |
| `--db`, `-d` | Target a specific database in the profile (defaults to first) |

---

## AI Agent Integration

### Non-interactive usage

Set the master password via environment variable so agents don't need a TTY:

```bash
export QBRIDGE_PASSWORD="your-master-password"
qbridge query --profile readonly "SELECT count(*) FROM orders"
```

### Example: Read-only analyst profile

```bash
# Register two databases
qbridge db add prod-postgres
qbridge db add analytics-mysql

# Create a read-only profile spanning both
qbridge profile add analyst
# Databases: prod-postgres,analytics-mysql
# Allow: SELECT

# Agent queries analytics DB
qbridge query --profile analyst --db analytics-mysql "SELECT * FROM daily_stats"
```

### Example: Write-allowed profile (no DDL)

```bash
qbridge profile add app-writer
# Databases: prod-postgres
# Allow: SELECT,INSERT,UPDATE,DELETE
```

---

## Security Notes

- Credentials are encrypted with **AES-256-GCM**; the key is derived from your master password using **Argon2id** (64 MB, 2 passes, 4 threads).
- The store file (`~/.qbridge/store.enc`) is unreadable without the master password.
- SQL classification uses keyword-level parsing — it is not possible to smuggle a `DELETE` inside a `SELECT` query.
- Profiles cannot be bypassed at the CLI level; all queries go through the enforcer.

---

## Supported Operations

| Operation | Covers |
|---|---|
| `SELECT` | `SELECT`, `WITH` (CTEs) |
| `INSERT` | `INSERT` |
| `UPDATE` | `UPDATE` |
| `DELETE` | `DELETE` |
| `DDL` | `CREATE`, `DROP`, `ALTER`, `RENAME`, `TRUNCATE` |
