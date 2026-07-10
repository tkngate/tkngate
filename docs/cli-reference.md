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

Once the server is running, the embedded React Telemetry Dashboard is accessible at `http://127.0.0.1:7478` (or your configured telemetry host/port).

## Config & Security

### `tkngate generate-master-key`
Generates a cryptographically secure 32-byte master key and **automatically saves it to `.env`**. You can also access this from the interactive menu under `config > Generate Master Key`.
```bash
tkngate generate-master-key

# Output:
# ┌─ New Master Key Generated ──────────┐
# |  [YOUR 32-BYTE HEX KEY]             |
# └──────────────────────────────────────┘
# SUCCESS Automatically saved Master Key to .env
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

## AI Web Application Firewall (WAF) & ZK-SNARKs

Tkngate includes a built-in pre-flight AI-WAF to intercept prompt injections, redact PII, and verify ZK-SNARK attestations from clients.

### `tkngate waf status`
Displays the current status of the AI-WAF engine, whether it is enabled in `tkngate.yaml`, and summarizes active rule sets.
```bash
tkngate waf status
```

### `tkngate waf rules`
Lists all active detection rules, including known prompt injection signatures, custom regex blocklists from `tkngate.yaml`, and the active PII redaction (DLP) categories.
```bash
tkngate waf rules
```

### `tkngate waf prove`
Generates a zero-knowledge proof (ZK-SNARK) attesting that a specific prompt does *not* violate any server-side blacklisted patterns. This is intended to be used by clients running a local Tkngate daemon.
```bash
tkngate waf prove "Translate this text to French."

# Output:
# X-Tkngate-ZKP: <base64-proof>:<base64-nonce>
```

### `tkngate waf verify`
Mathematically verifies a ZK-SNARK proof header (`X-Tkngate-ZKP`) without needing to see the original prompt. Useful for debugging rejected requests.
```bash
tkngate waf verify "<base64-proof>:<base64-nonce>"
```
