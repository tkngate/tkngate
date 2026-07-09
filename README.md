<p align="center">
  <img src="./public/tkngatecli.png" alt="Tkngate Banner" width="100%">
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

# 3. Start the interactive CLI
./tkngate
```

The CLI will guide you through generating a secure Master Key (AES-256) if you don't have one, and present a professional interactive menu to start the proxy, manage budgets, or view mesh telemetry.

---

## Enterprise Features

- **Virtual Budgets (Virtual Keys):** Issue `tkngate-sk-...` keys to your agents with hard USD caps. If a loop goes rogue, the proxy instantly blocks traffic, saving your credit card.
- **Semantic Caching:** Identical prompts are served from memory for `$0.00` with zero latency.
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
