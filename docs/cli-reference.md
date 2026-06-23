# Tkngate CLI Reference

The `tkngate` command-line interface provides everything you need to operate the reverse proxy, manage budgets, and generate secure credentials. It features an interactive, loop-based menu system for continuous management.

## Core Commands

### `tkngate` (Interactive Menu)
Running the binary without arguments starts the continuous interactive dashboard.
```bash
tkngate
```
From here, you can start the server, check budget status, view mesh pool metrics, and manage keys without leaving the terminal.

### `tkngate serve`
Starts the reverse proxy daemon and the telemetry API server directly.
```bash
tkngate serve
```
*Note: If `TKNGATE_MASTER_KEY` is not set in your environment, the CLI will interactively guide you to generate one instead of crashing.*

## Config & Security

### `tkngate config generate-master-key`
Generates a cryptographically secure 32-character master key. You must set this key in your environment to enable the zero-knowledge mesh encryption.
```bash
tkngate config generate-master-key

# Output:
# TKNGATE_MASTER_KEY="a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
```

## Virtual Key Management

Virtual Keys (`tkngate-sk-...`) are what you distribute to your AI agents or downstream clients instead of your raw OpenAI/Anthropic keys.

### `tkngate auth issue`
Generates a new Virtual Key and allocates a strict USD budget to it. Once this budget is exhausted, the proxy will return `429 Too Many Requests` for this key.
```bash
tkngate auth issue "Marketing_Agent" 10.50

# Output:
# Success! Virtual Key created:
# Key Name: Marketing_Agent
# Budget: $10.50
# Secret Key: tkngate-sk-x7y8z9... (Copy this now, it won't be shown again!)
```

### `tkngate auth list`
Displays a table of all active Virtual Keys, their allocated budgets, and how much they have consumed.
```bash
tkngate auth list
```

### `tkngate auth revoke`
Permanently deletes a Virtual Key and immediately blocks any further requests using it.
```bash
tkngate auth revoke "Marketing_Agent"
```

## Budget & Ledger

### `tkngate budget reset`
Wipes the entire SQLite transaction ledger and resets all consumed budgets to `$0.00`. Use this at the start of a new billing cycle.
```bash
tkngate budget reset
```

## P2P Token Mesh

### `tkngate pool donate`
Donates an API key to the decentralized DRR Token Mesh. The key is encrypted locally using your `TKNGATE_MASTER_KEY` before it is stored in the local SQLite ledger.

To prevent shell history leakage, if you do not provide the `--key` flag, the CLI will prompt you with a secure masked input (`****`).

```bash
# Secure interactive mode
tkngate pool donate "openai"

# Or scriptable mode (Warning: may log in bash history)
tkngate pool donate "openai" --key "sk-proj-YOUR_EXTRA_KEY"
```
