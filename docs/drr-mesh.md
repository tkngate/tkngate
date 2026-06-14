# DRR Token Mesh

The Deficit Round Robin (DRR) Token Mesh is Tkngate's peer-to-peer key sharing system — the first of its kind in the AI proxy space.

## How It Works

1. Developers donate spare API keys to the Tkngate mesh via `tkngate pool donate`.
2. Each key is encrypted with AES-256-GCM using the operator's `TKNGATE_MASTER_KEY` before being persisted to SQLite.
3. When an incoming request encounters a rate-limit or outage on the primary key, the DRR engine automatically rotates to a donated key from the mesh.

## Fairness Engine (BitTorrent Model)

To prevent abuse, Tkngate enforces a **Token Bucket** per session:

- **Free-riders** (sessions that haven't donated) are capped at **10,000 tokens/hour** from the shared pool.
- **Donors** receive priority access proportional to their contribution.

The DRR scheduler ensures no single donated key is over-utilised by tracking per-key deficits and rotating evenly.

## Key Security

All donated keys are stored as AES-256-GCM ciphertexts. The master key is loaded exclusively from the `TKNGATE_MASTER_KEY` environment variable and is **never written to disk**. Even if the SQLite database is exfiltrated, the keys are cryptographically unreadable.

## Stake-and-Slash Reputation

To prevent malicious nodes from abusing donated keys (e.g. sending ToS-violating prompts), Tkngate utilizes a **Stake-and-Slash Reputation System**:

1. **Trust Tiers**: Nodes start at a `NEW` tier. By consistently sending clean requests, they build up their trust score and advance to `TRUSTED` and `PREMIUM` tiers.
2. **Pre-flight Moderation**: Requests routed to the mesh are preemptively checked against OpenAI's `/v1/moderations` endpoint. If flagged for harmful content, the request is instantly dropped before it reaches the mesh key.
3. **Fraud Proofs**: Nodes can submit cryptographic `FraudProofs` if they receive malicious traffic. Tkngate validates these proofs and aggressively slashes the trust score of the offending node, eventually blacklisting them if they hit the `UNTRUSTED` tier.
