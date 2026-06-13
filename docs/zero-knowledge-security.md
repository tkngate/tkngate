# Zero-Knowledge Security

Tkngate implements a zero-knowledge encryption architecture to guarantee that donated API keys remain safe — even in a fully open-source deployment.

## Architecture

```
┌─────────────────────────┐
│  TKNGATE_MASTER_KEY     │  ← Env var (never on disk)
│  (32-byte AES-256 key)  │
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  AES-256-GCM Encrypt    │  ← Happens in RAM only
└────────────┬────────────┘
             │
             ▼
┌─────────────────────────┐
│  SQLite (budget.db)     │  ← Stores only ciphertext
│  token_pool_nodes table │
│  blinded_key_hash col   │
└─────────────────────────┘
```

## How It Works

1. The operator sets `TKNGATE_MASTER_KEY` as an environment variable (exactly 32 characters).
2. When a key is donated via `tkngate pool donate`, it is encrypted in memory using AES-256-GCM with a random nonce.
3. Only the hex-encoded ciphertext is written to SQLite. The plaintext key never touches disk.
4. At runtime, the DRR engine decrypts keys in-memory to attach them to outbound requests.

## If The Database Is Stolen

An attacker who exfiltrates `budget.db` will find only AES-256-GCM ciphertexts in the `blinded_key_hash` column. Without the master key (which lives in the environment, not on disk), the API keys are mathematically unrecoverable.

## Boot Requirements

```bash
export TKNGATE_MASTER_KEY="your-32-character-secret-key-here"
./tkngate serve
```

If the environment variable is missing or the wrong length, the proxy will refuse to start.
