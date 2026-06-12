# tkngate (Token Gateway)

`tkngate` is an advanced, open-source Layer-7 reverse proxy designed to protect, compress, and distribute API traffic for autonomous LLM agents (OpenAI, Anthropic). It is written in Go and backed by a local SQLite ledger.

## Core Features

### 🚦 Stateful Budgeting & Circuit Breakers (v0.1.0)
Stop runaway agents from burning through your wallet.
- **Session Tracking**: Pass the `X-Tkngate-Session-ID` header and track token spend per autonomous agent.
- **Agent Circuit Breakers**: If an agent hits its strict session budget limit, `tkngate` physically severs the connection (`HTTP 429`).
- **Dynamic Model Downgrade**: If an agent enters the "Amber Zone" (e.g., 75% of budget used), `tkngate` silently mutates the JSON payload on-the-fly, swapping expensive models (like `gpt-4o`) for cheaper fallbacks (like `gpt-4o-mini`).

### 🗜️ Context Compressor Engine (v0.2.0)
Stop "Token Maxxing" and save money structurally.
- **AST Pruning**: If an outbound request payload exceeds your `soft_token_limit`, `tkngate` parses the source code within the prompt using a pure-Go AST parser.
- **Lossless & Condensation**: It strips all non-functional comments and zeroes out deep function bodies while preserving the structural `{}` shell and method signatures—retaining the logic for the LLM while drastically reducing token count.

### 🌊 Local P2P Token Pool & DRR Routing (v0.3.0)
Evade rate limits by load-balancing across multiple API keys securely.
- **Blind Encryption**: Donate keys via the CLI. They are instantly encrypted using `AES-256-GCM` via a local, auto-generated master key.
- **Deficit Round Robin**: The proxy intercepts outbound traffic and uses a DRR engine to seamlessly rotate the `Authorization: Bearer` headers across all your donated keys.

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

## License
Apache 2.0
