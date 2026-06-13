# tkngate (The Cloudflare for AI Agents)

`tkngate` is an advanced, open-source Layer-7 reverse proxy designed to protect, compress, and distribute API traffic for autonomous LLM agents (OpenAI, Anthropic). Built for the enterprise, it acts as a global safety net for your AI workforce.

## Core Features

###  Stateful Budgeting & Circuit Breakers (v0.1.0)
Stop runaway agents from burning through your wallet.
- **Session Tracking**: Pass the `X-Tkngate-Session-ID` header and track token spend per autonomous agent.
- **Agent Circuit Breakers**: If an agent hits its strict session budget limit, `tkngate` physically severs the connection (`HTTP 429`).
- **Dynamic Model Downgrade**: If an agent enters the "Amber Zone" (e.g., 75% of budget used), `tkngate` silently mutates the JSON payload on-the-fly, swapping expensive models (like `gpt-4o`) for cheaper fallbacks (like `gpt-4o-mini`).

###  Context Compressor Engine (v0.2.0)
Stop "Token Maxxing" and save money structurally.
- **AST Pruning**: If an outbound request payload exceeds your `soft_token_limit`, `tkngate` parses the source code within the prompt using a pure-Go AST parser.
- **Lossless & Condensation**: It strips all non-functional comments and zeroes out deep function bodies while preserving the structural `{}` shell and method signatures—retaining the logic for the LLM while drastically reducing token count.

###  Local P2P Token Pool & DRR Routing (v0.3.0)
Evade rate limits by load-balancing across multiple API keys securely.
- **Blind Encryption**: Donate keys via the CLI. They are instantly encrypted using `AES-256-GCM` via a local, auto-generated master key.
- **Deficit Round Robin**: The proxy intercepts outbound traffic and uses a DRR engine to seamlessly rotate the `Authorization: Bearer` headers across all your donated keys.

###  Semantic Caching Layer (v0.4.0)
Instantly save 100% of the cost of duplicate agent requests.
- **Canonical Hashing**: The proxy normalizes incoming JSON payloads and creates a SHA-256 hash of the `model` and `messages`. 
- **Instant Hit**: If a payload is identical to a previous one, it intercepts the request before it leaves your machine and returns the cached JSON response for `$0.00` and `0ms` latency.
- **LRU & TTL Engine**: Thread-safe memory cache that automatically evicts stale or overflowing data.

###  Embedded Telemetry API (v0.5.0)
Monitor your proxy fleet safely.
- **Native Go API**: Runs a secondary REST API directly inside the proxy daemon alongside the reverse proxy.
- **Real-time Metrics**: Fetches instant, lock-free analytics from the SQLite ledger.
- **CORS-Ready Endpoints**: Exposes `/api/v1/overview`, `/api/v1/sessions`, and `/api/v1/pool` for generic telemetry scraping or integration into your own monitoring stack.

###  Polyglot Context Compressor (v0.6.0)
The Context Compressor engine now supports Python and JavaScript/TypeScript without requiring any CGO/C++ compiler toolchains.
- **Pure-Go Custom Lexers**: Instead of relying on heavy AST parsers, `tkngate` uses ultra-fast heuristic scanners.
- **Bracket & Indentation Tracking**: Safely counts JS braces and tracks Python indentation to structurally slice massive function bodies down to empty shells, preserving the signature for the LLM while dropping thousands of implementation tokens.

###  Seamless Auto-Retry Engine (v0.7.0)
Achieve "Cloudflare-level" reliability for autonomous agents.
- **429 Interception**: If an outbound request to OpenAI hits a `429 Too Many Requests` error, `tkngate` intercepts the failure before it crashes your agent.
- **Live Key Rotation**: It instantly pulls a fresh, donated API key from the DRR mesh pool and automatically retries the request on the fly.
- **Zero-Downtime**: The agent never knows a failure occurred, experiencing only a minor latency bump while the request successfully completes.

###  AI-WAF & DLP Engine (v0.8.0)
Enterprise-grade security and Data Loss Prevention (DLP) for LLM traffic.
- **Prompt Injection Firewall**: Scans outbound JSON payloads for known jailbreak vectors (e.g., "ignore all previous instructions"). Blocks malicious payloads with an `HTTP 403 Forbidden` before they reach OpenAI.
- **Auto-Redaction (PII)**: Uses lightning-fast heuristic regex to automatically detect and redact Credit Card numbers, SSNs, and leaked API keys (`sk-...`) from prompts, replacing them with `[REDACTED]`.

###  Universal API Router (v0.9.0)
Achieve 100% agent uptime with multi-model fallback routing across OpenAI-compatible providers.
- **Severe Outage Interception**: If OpenAI has a global outage (HTTP 500, 502, 503), the proxy intercepts the failure.
- **Auto-Translation**: It dynamically rewrites the destination URL, swaps `gpt-4o` for `deepseek-chat` or `moonshot-v1-8k` inside the JSON payload, grabs a new API key from the DRR mesh, and seamlessly fulfills the request.
- **Native Support**: Full local DRR support and active key validation for `openai`, `anthropic`, `deepseek`, `kimi`, and `groq`.

## Usage

1. Copy `tkngate.example.yaml` to `tkngate.yaml` and configure your API keys and budgets.
2. Run `go build` to build the `tkngate` executable.
3. Start the proxy with `./tkngate serve`.

### Commands

**Proxy & Budget Management:**
- `tkngate serve`: Starts the proxy daemon.
- `tkngate config show`: Prints resolved configuration (with secrets masked).
- `tkngate budget status`: Shows current spend vs. limits for each provider.
- `tkngate budget reset`: Manually resets the global budget ledger.

**Token Pool Management:**
- `tkngate pool donate --provider openai --key sk-... --limit 50000`: Safely encrypts and adds a key to the DRR rotation pool.
- `tkngate pool status`: Shows the total number of encrypted keys in your local mesh.

**Semantic Cache Management:**
- `tkngate cache status`: Displays live hits, misses, hit rate %, and estimated USD saved by the semantic cache.

## License
Apache 2.0
