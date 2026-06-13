# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
