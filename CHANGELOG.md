# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v2.2.0] - AI-WAF & ZKP Dashboard Visualization
### Added
- **Security Dashboard Tab**: The embedded React telemetry dashboard (served at `:7478`) now includes a dedicated "Security" tab to visualize AI-WAF rules and ZK-SNARK Attestations.
- **WAF Intercept Telemetry**: The UI now tracks `WAF Status` and `Total Intercepts` in real-time, pulling directly from the new `RawWafBlocks` atomic telemetry counter.
- **ZKP Enforcement Telemetry**: Displays whether `Strict ZKP Mode` is ENFORCED or OPTIONAL, and tracks valid proofs vs. rejected attestation attempts.
- **Bundled Execution**: The React frontend (`tkngate-dashboard`) is now pre-compiled and fully embedded into the Go binary (`tkngate.exe`) via `go:embed`. Running `tkngate serve` spins up the dashboard with zero external dependencies.

## [v2.1.0] - ZK-SNARK AI-WAF Attestations
### Added
- **Zero-Knowledge Proof Engine**: Integrated `gnark` (Groth16 BN254) to generate and verify ZK-SNARKs natively in Go. Clients can now cryptographically prove a prompt is safe without revealing the prompt's content.
- **WafCircuit**: A new `internal/zkp/circuit.go` defines the arithmetic circuit that attests a prompt hash does not match any blacklisted malicious prompt hashes.
- **ZKP Engine**: `internal/zkp/engine.go` handles circuit compilation, trusted setup, proof generation (`GenerateProof`), and verification (`VerifyProof`).
- **Strict ZKP Mode**: Added `mesh.strict_zkp_mode` config flag. When enabled, the DRR Mesh requires a valid ZK-SNARK proof before routing traffic through donated keys. Invalid proofs result in automatic reputation slashing.
- **`VerifyAndRoute`**: New DRR entry point in `internal/pool/drr.go` that enforces ZKP verification before key routing.
- **Anti-Replay Nonce**: Each proof is bound to a random attestation nonce, preventing proof reuse across requests.
- **Unit Tests & Benchmarks**: `circuit_test.go` verifies safe prompts pass, blacklisted prompts fail, and includes `BenchmarkProofGeneration`.

### Dependencies
- Added `github.com/consensys/gnark v0.15.0`
- Added `github.com/consensys/gnark-crypto v0.20.1`

## [v2.0.1] - Multi-Tenant Organizations & RBAC
### Added
- **Organizations**: Created the `tkngate_organizations` SQLite ledger table. You can now create specific budgets for individual teams or projects using `tkngate org create`.
- **Role-Based Access Control (RBAC)**: Virtual Keys can now be restricted to specific upstream providers (e.g. `openai` only) via the new `--providers` flag.
- **CLI Commands**: Added `tkngate org create` and `tkngate org list` commands, and updated `tkngate auth generate` to support the new `--org` and `--providers` flags.

### Fixed
- **API Server Registration**: Fixed a bug where `internal/api/server.go` failed to compile due to missing arguments in `budget.GlobalLedger.RegisterVirtualKey`.
- **Model Syncer**: Updated token counting models and defaults from `gpt-6o` to `gpt-5.6` to reflect the latest release.

## [v2.0.0] - Stake-and-Slash Reputation Engine
### Added
- **Game-Theory Economics**: Fully wired the Stake-and-Slash Reputation Engine into the live proxy pipeline.
- **Preflight Moderation**: Automatically blocks ToS-violating prompts using OpenAI's moderation API *before* they consume donated keys.
- **WAF Slashing Hook**: Triggering AI-WAF jailbreak blocks now permanently degrades the offending node's mesh trust score.
- **Tier-Based Routing**: The DRR Fairness Engine now prevents `UNTRUSTED` and `NEW` nodes from routing traffic through enterprise-grade, high-TPM premium keys, protecting them from abuse.
- **Mesh Telemetry API**: Added `/api/v1/mesh/reputation` endpoint to observe real-time trust scores.
- **Prometheus Metrics**: `tkngate_mesh_slashes_total` and `tkngate_mesh_trust_score` now exported.

## [v1.9.4] - Tool-Calling & Structured Output Support
### Fixed
- **Semantic Cache Corruption**: The Semantic Cache now deterministically isolates JSON payload components (`tools`, `tool_choice`, `response_format`) so that Agentic requests (e.g. LangChain, AutoGen) never collide with plain-text cache keys, completely preventing hallucinated tool-call returns.

## [v1.9.3] - Prometheus & OpenTelemetry Metrics
### Added
- **Prometheus Exporter**: The `/metrics` endpoint on the telemetry port now exports real-time enterprise metrics (`tkngate_budget_spent_usd_total`, `tkngate_active_connections`, `tkngate_virtual_key_spend_usd_total`) allowing instant integration with Datadog and Grafana.

## [v1.9.2] - Distributed Redis Caching
### Added
- **Distributed Semantic Caching**: The Semantic Cache can now be globally shared across all Tkngate nodes by pointing `cache.redis_uri` to a Redis cluster.
- **CLI Commands**: Added `tkngate cache status` to retrieve cache savings and hit rates, and `tkngate cache clear` to flush the cache.

## [v1.9.1] - Dynamic WAF Rules & CLI Fixes
### Added
- **Dynamic AI-WAF**: The Web Application Firewall (WAF) now supports custom regex blocklists defined in `tkngate.yaml`. You can now block proprietary project names, credit cards, or custom prompt-injection signatures at the proxy level.
### Fixed
- **Startup Crash**: Fixed a nil pointer dereference in the logging engine during WAF initialization if telemetry was disabled.

## [v1.9.0] - Enterprise SDKs & Critical Stability Fixes
### Added
- **Official SDK Wrappers**: Zero-friction drop-in SDK wrappers for Python (`tkngate`) and Node.js (`tkngate`). 
- **Tests**: Introduced `httptest` unit tests for core proxy middleware to guarantee failovers and limits under load.

### Fixed
- **Semantic Cache Canonicalization**: Fixed a bug where identical prompts missed the cache if stochastic parameters like `temperature` were changed. The cache now correctly extracts and hashes only the `messages` array.
- **SQLite Database Locking**: Enabled Write-Ahead Logging (WAL) mode in the local ledger to resolve `database is locked` errors during high-concurrency agent loops.
- **Streaming Telemetry Accuracy**: Fixed an issue where TTFT (Time To First Token) and Latency were reported as 0ms for streaming responses.

### Changed
- **Fairness Engine Configuration**: The Deficit Round Robin free-rider limit is no longer hardcoded at 10,000 tokens. It is now configurable via `tkngate.yaml`.
- **Wedge Pitch**: Updated project documentation to emphasize the Enterprise Budget Firewall and Semantic Caching capabilities over the P2P Mesh.

## [v1.8.0] - Branding & Assets 
### Added
- **TrueColor Branding**: Migrated CLI output to fully support 24-bit TrueColor RGB styles,"Vintage Brutalist" aesthetic (#b89752 Gold and #162b1d Forest Green).
### Fixed
- **WAF Regex Order Issue**: Resolved a non-deterministic map iteration bug in the `RedactPII` WAF engine that misclassified JSON Web Tokens (JWT) as generic environment secrets.
- **Telemetry Auth Mismatch**: Aligned the local dashboard telemetry endpoints to strictly expect a 32-character `TKNGATE_MASTER_KEY` Bearer token.

## [v1.7.0] - Enterprise Observability & Launch Readiness
### Added
- **Prometheus Metrics**: Native `/metrics` endpoint exposing `tkngate_requests_total`, `tkngate_tokens_consumed_total`, `tkngate_cache_hits_total`, and `tkngate_waf_intercepts_total` for Grafana/Datadog integration.
- **Dockerfile**: Multi-stage production container image (Alpine-based, under 25MB).
- **Docker Compose**: Full stack (`tkngate` + `redis`) single-command deployment.
- **SDK Examples**: Drop-in Python and Node.js examples showing one-line integration with the OpenAI SDK.

## [v1.6.1] - Generate Master Key CLI
### Added
- **Master Key Generator**: Added `tkngate config generate-master-key` CLI command to easily generate secure 32-character AES-256-GCM encryption keys.

## [v1.6.0] - Stake-and-Slash Mesh Reputation
### Added
- **Reputation Ledger**: Added SQLite-backed trust scoring (`NEW`, `TRUSTED`, `PREMIUM`, `UNTRUSTED`).
- **Pre-flight Moderation**: Requests routed to mesh nodes are preemptively checked via OpenAI's `/v1/moderations` to protect donated keys from abuse.
- **Fraud Proofs**: Added cryptographic `SubmitFraudProof` pipeline to slash and blacklist malicious senders.
- **Trust-Tier Routing**: Integrated reputation scores directly into the Deficit Round Robin (DRR) engine.

## [v1.5.0] - Strict Rate Limiting
### Added
- **Token Bucket Middleware**: Integrated `golang.org/x/time/rate` for highly-performant, in-memory rate limiting.
- **Agent Abuse Protection**: Configurable RPM and burst limits dynamically protect upstream providers from autonomous agent loops.

## [v1.4.0] - Distributed Semantic Caching
### Added
- **Redis Engine Integration**: Swapped local LRU cache for a distributed Redis-backed cache for horizontal scalability.
- **Fleet-Wide Telemetry**: Centralized hits, misses, and savings metrics via atomic Redis counters.

## [v1.3.0] - Enterprise Virtual Keys
### Added
- **Virtual Auth Layer**: Issued `tkngate-sk-...` virtual keys, shifting from header-based trust to true authentication.
- **CLI Management**: Added `tkngate auth list` and `tkngate auth revoke`.

## [v1.2.0] - Enterprise Resiliency
### Added
- **SSE Streaming Support**: Native Server-Sent Events interception with real-time chunk token counting and mid-stream budget cutoffs.
- **Enhanced PII/DLP Engine**: Added typed redaction markers (`[REDACTED_EMAIL]`, `[REDACTED_AWS_KEY]`) for Emails, Phones, AWS Keys, GitHub Tokens, JWTs, and Private Keys.
- **Universal API Router Expansion**: Added `anthropic` and `openai` as proper smart fallback providers with automatic payload and header translation.

## [v1.1.1] - CLI UI Update
### Added
- Beautiful terminal output using `fatih/color` for `tkngate config show`.
- Stunning ASCII art startup banner for `tkngate serve`.

## [v1.1.0] - Shadow Mode & 2026 Models
### Added
- Enterprise Shadow Mode: Traffic mirroring for evaluating alternative models risk-free.
- Sync with latest 2026 AI models (`gpt-6o`, `gpt-5.5-turbo`, `claude-4.8-opus`, `deepseek-chat-v3`).
- Security hardening and Denial of Service (DoS) protection patches.

## [v1.0.0] - Enterprise Core
### Added
- Zero-Knowledge Security: `TKNGATE_MASTER_KEY` environment variable enforcement.
- BitTorrent-style Fairness Engine: Deficit Round Robin limits for free-riders.
- Local Dashboard API endpoint `/api/v1/mesh/stats` for real-time mesh capacity visualization.

## [v0.9.0] - Universal API Router
### Added
- Dynamic JSON body mutation for cross-provider fallback (e.g. `gpt-4o` to `deepseek`).
- Support for `kimi` and `groq` provider API verification endpoints.

## [v0.8.0] - AI-WAF & DLP
### Added
- Web Application Firewall specifically tuned for LLM prompt injection signatures.
- Data Loss Prevention (DLP) to auto-redact SSNs, Credit Cards, and API keys (`[REDACTED]`).

## [v0.7.0] - Auto-Retry Engine
### Added
- Zero-Downtime fallback mechanism for 500, 502, 503 HTTP status codes.
- Exponential backoff loop configured dynamically by proxy director.

## [v0.6.0] - Polyglot Context Compressor
### Added
- Polyglot Context Compressor with custom Go heuristic lexers.
- Support for Python indentation tracking and semantic compression.
- Support for JavaScript/TypeScript bracket-counting semantic compression.

## [v0.5.0] - Embedded Telemetry API
### Added
- Embedded Go REST API running natively inside the proxy.
- `/api/v1/overview` endpoint for global metrics and cache stats.
- `/api/v1/sessions` endpoint for agent tracking.
- `/api/v1/pool` endpoint for checking the P2P key mesh.
- Configuration block for telemetry inside `tkngate.yaml`.

## [v0.4.0] - Semantic Caching Layer
### Added
- Local memory LRU Cache engine for LLM JSON payloads.
- SHA-256 canonical hashing of `model` and `messages`.
- Zero-latency proxy cache hit pipeline.
- `tkngate cache status` CLI command.

## [v0.3.0] - Local P2P Token Pool
### Added
- Blind Encryption engine using `AES-256-GCM`.
- Deficit Round Robin (DRR) request router.
- `tkngate pool donate` CLI command.

## [v0.2.0] - Context Compressor Engine
### Added
- Go AST parsing for LLM payloads.
- Comment stripping and deep function block omissions.

## [v0.1.0] - Stateful Budgeting
### Added
- SQLite ledger integration.
- `X-Tkngate-Session-ID` middleware processing.
- Global and Session-based circuit breakers.
- Dynamic Amber Zone model downgrading.
