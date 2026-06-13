<p align="center">
  <h1 align="center"> Tkngate</h1>
  <p align="center"><strong>The Cloudflare for Autonomous AI Agents</strong></p>
  <p align="center">
    Zero-knowledge reverse proxy · Multi-provider failover · P2P token mesh
  </p>
</p>

<p align="center">
  <a href="#install">Install</a> •
  <a href="#why-tkngate">Why Tkngate</a> •
  <a href="#features">Features</a> •
  <a href="#configuration">Configuration</a> •
  <a href="./docs">Docs</a>
</p>

---

## Why Tkngate

Your autonomous agents crash when OpenAI goes down. Your budget bleeds when a rogue loop burns $200 in tokens. Your API keys sit in plain text in `.env` files.

**Tkngate fixes all of this.** It sits between your agents and the AI providers, acting as an intelligent shield that manages keys, budgets, security, and failover — automatically.

## Install

```bash
git clone https://github.com/tkngate/tkngate.git
cd tkngate
cp tkngate.example.yaml tkngate.yaml   # edit with your keys
export TKNGATE_MASTER_KEY="your-32-char-secret-key-here-!!"
go run main.go serve
```

Your agents now point to `http://localhost:7477/openai/chat/completions` instead of `api.openai.com`.

## Features

### Universal API Router
Route through **OpenAI, Anthropic, DeepSeek, Kimi, and Groq** from a single endpoint. If one provider goes down (HTTP 500/502/503), Tkngate automatically fails over to the next.

### AI-WAF & DLP
Block prompt injection attacks and automatically redact PII (credit cards, SSNs, API keys) before they reach the provider.

### Budget Traffic Lights
Real-time spend tracking with Green → Amber → Red zones. Set global limits, per-session caps, and automatic request blocking when budgets are exhausted.

###  Semantic Cache
Identical prompts are served from an in-memory cache, saving tokens and money. Cache keys are computed from normalised `model + messages` hashes.

### 🗜️ Context Compressor
Automatically compresses Go, Python, and JavaScript code blocks in prompts — stripping comments and whitespace to reduce token usage by up to 40%.

### 🌐 P2P Token Mesh (DRR)
The world's first BitTorrent-style token pool for LLM APIs. Donate spare API keys to the mesh, and get priority access to the network's capacity during outages. Protected by AES-256 zero-knowledge encryption.

### 👻 Shadow Mode
Silently mirror a fraction of production traffic to an alternative provider (e.g., DeepSeek) to evaluate cost savings — with zero latency impact on the primary request.

## Configuration

See [`tkngate.example.yaml`](./tkngate.example.yaml) for the full reference.

## Documentation

| Topic | Link |
|-------|------|
| Budget System & Traffic Lights | [docs/budgeting.md](./docs/budgeting.md) |
| DRR Token Mesh | [docs/drr-mesh.md](./docs/drr-mesh.md) |
| Zero-Knowledge Security | [docs/zero-knowledge-security.md](./docs/zero-knowledge-security.md) |
| Shadow Mode | [docs/shadow-mode.md](./docs/shadow-mode.md) |

## Architecture

```
Agent Request
     │
     ▼
┌──────────────────────────────────────┐
│            TKNGATE PROXY             │
│                                      │
│  Budget Guard → AI-WAF/DLP           │
│       → Context Compressor           │
│       → Semantic Cache               │
│       → Auto-Retry (3x)             │
│       → Universal Router (Failover)  │
│       → Shadow Mode (async mirror)   │
│       → Token Counter → Ledger       │
└──────────────────────────────────────┘
     │
     ▼
  OpenAI / Anthropic / DeepSeek / Kimi / Groq
```

## License

Apache 2.0
