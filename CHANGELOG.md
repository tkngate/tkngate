# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
