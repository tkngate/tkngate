<p align="center">
  <img src="./public/tkngatecli.png" alt="Tkngate Banner" width="60%">
</p>

<h1 align="center">Tkngate: The P2P Token Mesh</h1>
<p align="center"><strong>The Cloudflare for Autonomous AI Agents</strong></p>
<p align="center">
  <a href="#install">Install</a> •
  <a href="#the-tragedy-of-the-commons-the-mesh">The Mesh</a> •
  <a href="#features">Features</a> •
  <a href="#configuration">Configuration</a> •
  <a href="./docs">Docs</a>
</p>

---

## What is Tkngate?

Tkngate is an enterprise-grade, zero-knowledge reverse proxy daemon that protects your autonomous agents and LLM budgets. It provides **Strict Rate Limiting**, **Semantic Caching**, and **Universal Failover** out of the box.

But its killer feature is the **Deficit Round Robin (DRR) Token Mesh**: *BitTorrent for LLM Tokens.*

If you run parallelized agents, you will quickly hit OpenAI's rate limits (`429 Too Many Requests`). Tkngate solves this by allowing developers and enterprises to safely pool their unused API keys into a decentralized, zero-knowledge mesh, instantly multiplying bandwidth for everyone.

---

## The Mesh: Stake-and-Slash

The problem with sharing API keys in a pool is trust. How do you know a rogue node won't use your clean OpenAI key to process ToS-violating prompts, getting your account banned? 

Tkngate solves this using an **Economic Game Theory Ledger**:

1. **AES-256 Zero-Knowledge Payload**: When you donate an API key, it is encrypted locally. You never see the plaintext prompts passing through your node.
2. **Pre-flight Moderation WAF**: Every prompt is scanned locally before being routed, protecting against prompt injections and PII leakage.
3. **The Slash**: If a malicious node somehow bypasses the WAF and OpenAI flags the request, the HTTP response acts as a **Fraud Proof**. The malicious sender's public key is permanently blacklisted globally, and their stake in the mesh is slashed to zero.

---

## Quick Start (Interactive CLI)

Tkngate is a single binary with zero external dependencies (no Redis or Postgres required out of the box). Version 1.7.0 introduces a robust, interactive CLI loop for managing your mesh and budgets seamlessly.

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
