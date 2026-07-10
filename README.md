<p align="center">
  <img src="./public/tkngatecli.png" alt="Tkngate Banner" width="50%">
</p>

<h1 align="center">Tkngate: The Enterprise Budget Firewall</h1>
<p align="center"><strong>The Cloudflare for Autonomous AI Agents</strong></p>
<p align="center">
  <a href="#install">Install</a> •
  <a href="#features">Features</a> •
  <a href="#configuration">Configuration</a> •
  <a href="./docs">Docs</a>
</p>

---

## What is Tkngate?

Tkngate is an enterprise-grade, zero-knowledge reverse proxy daemon that protects your autonomous agents and LLM budgets. Every developer building with LangChain, AutoGen, or CrewAI has experienced a runaway `while` loop that burns $50 in OpenAI credits in 10 minutes. 

Tkngate solves this by sitting between your agents and your provider (OpenAI, Anthropic, etc.), providing **Strict Rate Limiting**, **Virtual Keys with hard USD limits**, **Semantic Caching**, and a **Pre-flight WAF (Web Application Firewall)** out of the box.

* "Stop paying for 429s."
* "Put hard USD circuit breakers on your autonomous agents."
* "Cache redundant agent reasoning loops for $0.00."
* "Create Multi-Tenant Organizations and restrict keys using RBAC."
* "Prove your prompts are safe with Zero-Knowledge (ZK-SNARK) AI-WAF attestations."
* "Visualize AI-WAF intercepts and mesh capacity live on the built-in React Dashboard."

---

## Zero-Friction SDK Drop-ins

Tkngate is designed as a drop-in replacement for the official OpenAI SDKs. You don't need to rewrite your agent logic. Just change the `baseURL` to point to localhost:7477 and pass your Virtual Key. 

See the [`examples/`](./examples) directory for Python and Node.js integrations.

---

## Quick Start (Interactive CLI)

Tkngate is a single binary with zero external dependencies (no Redis or Postgres required out of the box). Version 1.9.0 introduces a robust, interactive CLI loop for managing your mesh and budgets seamlessly.

```bash
git clone https://github.com/tkngate/tkngate.git
cd tkngate

# 1. Build the binary
go build -o tkngate

# 2. Setup your config
cp tkngate.example.yaml tkngate.yaml 

### 3. Start the Daemon

```bash
tkngate serve
```

### 4. Telemetry & Observability

Tkngate natively exports Prometheus metrics on the telemetry port (default `7478`). You can hook this directly into Datadog, Grafana, or New Relic to monitor enterprise spend.

Add this job to your `prometheus.yml`:
```yaml
scrape_configs:
  - job_name: 'tkngate'
    static_configs:
      - targets: ['localhost:7478']
```

The exporter provides detailed metrics including:
- `tkngate_budget_spent_usd_total`: Total enterprise API cost.
- `tkngate_virtual_key_spend_usd_total`: API cost tracked by individual Virtual Keys.
- `tkngate_active_connections`: In-flight requests to the proxy.
- `tkngate_cache_hits_total`: Total cache hits.
- `tkngate_waf_intercepts_total`: Requests blocked by the AI-WAF.

---

## 🏗️ The Mesh (Decentralised Load Balancing)

As of `v2.0.0`, Tkngate features a full **Stake-and-Slash Reputation Engine** for peer-to-peer enterprise pooling. 
Instead of a single bottleneck API key, enterprise teams can donate multiple keys into a shared pool governed by Game Theory:

1. **Deficit Round Robin (DRR):** The mesh algorithm guarantees that nodes consuming too many tokens without donating their own keys (free-riders) are heavily throttled.
2. **Stake-and-Slash Reputation:** Nodes start with a base Trust Score. 
   - If they route safe, clean prompts, their score increases. 
   - If they route a prompt that gets flagged by our **Dynamic AI-WAF** or the **OpenAI Pre-flight Moderation Engine**, their trust is instantly slashed. 
   - If trust drops too low, the DRR automatically isolates them from routing through high-TPM, enterprise-grade premium keys, eliminating the "Tragedy of the Commons".

---

## Enterprise Features

- **Deficit Round Robin (DRR)** routing algorithm ensures completely fair bandwidth distribution across nodes.
- **Enterprise Budget Guard** sets hard limits on per-session and global token spend to prevent run-away AI agents.
- **Semantic Caching** saves you up to 80% on repetitive prompt token costs.
- **Distributed Redis Caching** allows multiple Tkngate nodes to share cached responses instantly across your load-balanced enterprise infrastructure.
- **Dynamic AI-WAF** intercepts prompt-injections, custom enterprise secrets, and ToS-violating keywords before they hit OpenAI.
- **Universal Fallback:** If OpenAI returns a `500` or `503`, Tkngate automatically transparently falls back to Anthropic or DeepSeek.
- **Context Compressor:** Automatically strips comments and whitespace from code-heavy prompts, reducing token bills by up to 40%.
- **Shadow Mode:** Silently mirror 10% of your production traffic to a cheaper provider (like DeepSeek) to evaluate responses without impacting your main flow latency.
- **Secure Telemetry API:** Monitor all nodes and sessions locally with Bearer-token authenticated metrics and CORS lockdown.

---

## Documentation

| Topic | Link |
|-------|------|
| CLI Reference & Architecture | [docs/cli-reference.md](./docs/cli-reference.md) |
| Budget System & Traffic Lights | [docs/budgeting.md](./docs/budgeting.md) |
| DRR Token Mesh & Reputation | [docs/drr-mesh.md](./docs/drr-mesh.md) |
| Enterprise Virtual Keys | [docs/virtual-keys.md](./docs/virtual-keys.md) |
| Strict Rate Limiting | [docs/rate-limiting.md](./docs/rate-limiting.md) |

## License
Apache 2.0
