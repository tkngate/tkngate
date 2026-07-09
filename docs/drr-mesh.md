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

## Stake-and-Slash Reputation Engine (v2.0.0)

To prevent malicious nodes from abusing donated keys (e.g., getting them rate-limited or banned by sending ToS-violating prompts), Tkngate utilizes a **Game-Theory Economics Engine**:

1. **Trust Tiers**: Nodes start at a `NEW` tier (50/100). By consistently sending clean requests, they build up their trust score to 100 and advance to `TRUSTED` and `PREMIUM` tiers.
2. **Pre-flight Moderation**: If enabled, requests routed to the mesh are preemptively checked against OpenAI's `/v1/moderations` endpoint. If flagged for harmful content, the proxy instantly drops the request, and the sender's Trust Score is **slashed by 25 points**.
3. **AI-WAF Slashing**: If the proxy's internal Web Application Firewall (WAF) detects a prompt-injection, jailbreak, or regex blocklist violation, the request is blocked, and the sender's Trust Score is **slashed by 25 points**.
4. **The Premium Gate**: If a node's trust score drops to `UNTRUSTED` (below 10/100) or they are still in the `NEW` tier, the DRR automatically isolates them. They are physically blocked from drawing enterprise-grade "Premium" keys (>10M TPM) from the pool until they rebuild their reputation. 

This guarantees the "Tragedy of the Commons" is solved: bad actors cannot drain or pollute the high-value keys in the mesh.
