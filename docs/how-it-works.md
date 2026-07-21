# How TknGate Works

`tkngate` is a high-performance Layer-7 reverse proxy written in Go, designed to run as a **Kubernetes sidecar** or standalone daemon inside your private VPC. Instead of your applications sending HTTP requests directly to OpenAI or Anthropic using raw API keys, they send requests to the `tkngate` sidecar.

`tkngate` isolates the credentials, runs a gauntlet of smart middleware (WAF, Budgeting, Caching), and then forwards the request to the upstream LLM provider.

## The Architecture Pipeline

When an HTTP request (like `/v1/chat/completions`) hits `tkngate`, it goes through the following lifecycle:

### 1. Identity & Session Extraction
The proxy looks for the `X-Tkngate-Session-ID` HTTP header. This header identifies *which* specific autonomous agent or user is making the request.

### 2. The Budget Guard (Circuit Breaker)
`tkngate` queries a lightning-fast local SQLite ledger to check how much money this specific session has spent.
- **Green Zone (OK)**: The request is allowed to continue.
- **Amber Zone (Warning)**: The request continues, but `tkngate` rewrites the JSON payload to downgrade the model (e.g., swapping `gpt-4o` for `gpt-4o-mini`) to save money.
- **Red Zone (Danger)**: `tkngate` physically drops the request, returning a `429 Too Many Requests` error back to the agent. This breaks infinite loops.

### 3. The Context Compressor Engine
If the request is allowed through, `tkngate` estimates the token count of the prompt payload using the `tiktoken` library.
- If the token count exceeds the `soft_token_limit`, the compressor activates.
- It parses the prompt looking for raw code (e.g., Go source code).
- It structurally strips out non-functional comments and zeroes out deep function bodies using an Abstract Syntax Tree (AST).
- It reserializes a heavily optimized, much smaller JSON payload.

### 4. The Deficit Round Robin (DRR) Router
Instead of using a single API key, `tkngate` pulls from a local encrypted pool of "donated" keys.
- It uses a DRR algorithm to mathematically load-balance the request across all available keys.
- It decrypts the chosen key using an AES-256-GCM master key and injects it into the `Authorization: Bearer` header.

### 5. Forward & Audit
The optimized request is sent to OpenAI. When the response comes back, `tkngate` counts the exact output tokens, calculates the exact USD cost, and asynchronously writes the transaction to the SQLite ledger before returning the response to the client.
