<p align="center">
  <img src="./public/tkngatecli.png" alt="Tkngate Banner" width="50%">
</p>

<h1 align="center">Tkngate: The Enterprise AI Gateway</h1>
<p align="center"><strong>The Cloudflare for Autonomous AI Agents</strong></p>
<p align="center">
  <a href="#install">Install</a> •
  <a href="#quick-start">Quick Start</a> •
  <a href="#why-tkngate">Why Tkngate?</a> •
  <a href="#features">Features</a> •
  <a href="#vs-the-competition">Comparison</a> •
  <a href="./docs">Docs</a>
</p>

---

## What is Tkngate?

Every developer building with **LangChain, AutoGen, or CrewAI** has experienced the same nightmare: a runaway `while` loop that burns $50 in OpenAI credits in 10 minutes. You wake up to a surprise bill with no way to explain which agent did it, when, or why.

**Tkngate solves this permanently.**

It is a single Go binary that sits as a reverse proxy between your agents and your LLM providers (OpenAI, Anthropic, Groq, etc.). It enforces hard USD circuit breakers, inspects every prompt with a ZK-verified AI-WAF, caches redundant reasoning loops for free, and gives you a real-time dashboard — all without sending a single byte of your data to a third party.

```
Your AI Agent  →  tkngate (localhost:7477)  →  OpenAI / Anthropic / Groq / DeepSeek
```

---

## Why Tkngate?

> *"LiteLLM routes. Portkey observes. **Tkngate protects.**"*

### The Problem With Every Other Tool

| Tool | The Gap |
|------|---------|
| **LiteLLM** | Routes between models. No hard budget limits. Requires Python. |
| **Portkey** | Cloud-only. Your prompts go through their servers. |
| **Helicone** | Observability only. Can't stop a runaway agent. |
| **OpenRouter** | You use their shared keys, not yours. |
| **Kong AI Gateway** | Enterprise-only. Requires existing Kong infrastructure. |

None of them offer **cryptographic prompt security**. None of them have a **P2P key mesh**. All of them either require a cloud account or send your data somewhere.

---

## vs. The Competition

| Feature | **Tkngate** | LiteLLM | Portkey | Helicone |
|---------|:-----------:|:-------:|:-------:|:--------:|
| Single binary (no Python, no Docker) | ✅ | ❌ | ❌ | ❌ |
| 100% self-hosted, no data leaves your machine | ✅ | ✅ | ❌ | ❌ |
| **ZK-SNARK AI-WAF attestations** | ✅ | ❌ | ❌ | ❌ |
| **P2P encrypted key mesh (DRR)** | ✅ | ❌ | ❌ | ❌ |
| Hard USD per-session circuit breakers | ✅ | ⚠️ soft | ⚠️ | ❌ |
| Shadow Mode A/B traffic splitting | ✅ | ❌ | ❌ | ❌ |
| Multi-tenant RBAC + Organizations | ✅ | ✅ | ✅ | ❌ |
| Semantic cache (save up to 80%) | ✅ | ❌ | ❌ | ❌ |
| Embedded real-time dashboard | ✅ | ❌ | ✅ (cloud) | ✅ (cloud) |
| Prometheus metrics export | ✅ | ✅ | ⚠️ | ⚠️ |

---

## What Makes Tkngate Unique

### 1. 🔐 Zero-Knowledge Proof Attestations — Nobody Else Has This
Every competitor routes prompts in plain text. Tkngate is the **only LLM proxy** that uses ZK-SNARKs (Groth16 BN254) to mathematically prove a prompt is safe **without ever reading the prompt content**. Enable it with one config flag:
```yaml
mesh:
  strict_zkp_mode: true
```

### 2. 📦 One Binary. Zero Dependencies.
```bash
go build -o tkngate && ./tkngate serve
```
That's it. No `pip install`. No Docker Compose. No cloud account. No data leaving your machine. SQLite is the only storage engine by default.

### 3. 🤝 Encrypted P2P Token Mesh
Tkngate lets you donate your spare API quota into a cryptographically secured pool. Keys are encrypted with **AES-256-GCM** and never exposed in plaintext, even to other nodes in the mesh. A **Deficit Round-Robin (DRR)** algorithm ensures free-riders are throttled automatically.

### 4. 💰 Hard USD Circuit Breakers — Not Soft Limits
Most tools let you *monitor* spend. Tkngate lets you *stop* it. The moment a session or global budget is breached, the connection is terminated — not after the next billing cycle, **right now, at the byte level**.

### 5. 👁️ Shadow Mode — A/B Test Any Two Models Silently
Fork a percentage of live traffic to a second provider and compare outputs and costs — with **zero changes to your agent code**. Like Canary Deployments for LLMs.

### 6. 🛡️ Live Security Dashboard
The built-in React dashboard (`http://localhost:7478`) shows real-time WAF intercepts, ZKP attestation counts, mesh capacity, budget zones, and per-key spend — all served from the binary itself.

---

## Quick Start

```bash
# Clone & build
git clone https://github.com/tkngate/tkngate.git
cd tkngate
go build -o tkngate

# Configure
cp tkngate.example.yaml tkngate.yaml
# Edit tkngate.yaml with your provider API keys

# Generate a master key (required for the dashboard)
./tkngate generate-master-key

# Run
./tkngate serve
```

The proxy starts on `localhost:7477`. The telemetry dashboard opens at `localhost:7478`.

Point your existing agent at tkngate — no code rewrites needed:

```python
# Python (OpenAI SDK)
from openai import OpenAI
client = OpenAI(
    api_key="your-tkngate-virtual-key",
    base_url="http://localhost:7477/v1"
)
```

```javascript
// Node.js (OpenAI SDK)
const openai = new OpenAI({
  apiKey: "your-tkngate-virtual-key",
  baseURL: "http://localhost:7477/v1"
});
```

---

## Features

- **Deficit Round Robin (DRR)** — fair, weighted routing across all nodes in the mesh
- **Enterprise Budget Guard** — hard USD limits per-session and globally, with GREEN/AMBER/RED zone alerts
- **Semantic Caching** — up to 80% savings on repeated or similar prompts (in-memory or Redis)
- **Dynamic AI-WAF** — intercepts prompt injections, PII leaks, and jailbreaks before they hit your provider
- **ZK-SNARK Attestations** — cryptographic proof of prompt safety using Groth16 BN254 circuits
- **Shadow Mode** — silently mirror a percentage of traffic to a second model for evaluation
- **Universal Fallback** — if OpenAI returns 5xx, auto-retry on Anthropic or DeepSeek transparently
- **Context Compressor** — strip whitespace and comments from code-heavy prompts (up to 40% token reduction)
- **Multi-Tenant RBAC** — Virtual Keys with per-org budgets and provider restrictions
- **Prometheus Export** — native metrics at `/metrics` for Grafana, Datadog, or New Relic
- **Embedded Dashboard** — real-time React UI served from the binary itself, no Node.js required
- **Stake-and-Slash Reputation** — mesh nodes that route malicious prompts get their trust score slashed automatically

---

## Telemetry & Observability

Tkngate natively exports Prometheus metrics at `http://localhost:7478/metrics`. Hook it into any observability stack:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'tkngate'
    static_configs:
      - targets: ['localhost:7478']
```

Key metrics:
- `tkngate_budget_spent_usd_total` — total enterprise API cost
- `tkngate_virtual_key_spend_usd_total` — spend per virtual key
- `tkngate_active_connections` — in-flight requests
- `tkngate_cache_hits_total` — semantic cache hits
- `tkngate_waf_intercepts_total` — requests blocked by the AI-WAF

---

## The P2P Mesh

As of `v2.0.0`, Tkngate features a full **Stake-and-Slash Reputation Engine** for peer-to-peer enterprise pooling. Instead of a single bottleneck API key, teams donate multiple keys into a shared pool governed by game theory:

1. **Deficit Round Robin (DRR):** Free-riders who consume without donating are automatically throttled.
2. **Stake-and-Slash Reputation:** Nodes route clean prompts → trust increases. Nodes route flagged prompts → trust is slashed. Low-trust nodes lose access to premium high-TPM keys.
3. **ZK-SNARK Verification:** In `strict_zkp_mode`, every key access requires a valid zero-knowledge proof, making it impossible to abuse the mesh.

---

## Documentation

| Topic | Link |
|-------|------|
| CLI Reference & Architecture | [docs/cli-reference.md](./docs/cli-reference.md) |
| Budget System & Traffic Lights | [docs/budgeting.md](./docs/budgeting.md) |
| DRR Token Mesh & Reputation | [docs/drr-mesh.md](./docs/drr-mesh.md) |
| Enterprise Virtual Keys | [docs/virtual-keys.md](./docs/virtual-keys.md) |
| Strict Rate Limiting | [docs/rate-limiting.md](./docs/rate-limiting.md) |
| AI-WAF & Security | [docs/cli-reference.md#waf](./docs/cli-reference.md) |

---

## License
Apache 2.0
