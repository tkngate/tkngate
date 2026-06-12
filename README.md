# tkngate (Token Gateway)

tkngate is an open-source token-management reverse proxy daemon written in Go. It forwards requests to LLM APIs (OpenAI/Anthropic), handles request/response token counting, and tracks budgets with a SQLite-backed ledger.

## Features

- **Reverse Proxy**: HTTP reverse proxy forwarding to OpenAI & Anthropic APIs.
- **Token Counting**: Count input/output tokens per request using `tiktoken-go`.
- **Budget Ledger**: SQLite-backed per-key spend tracking with configurable limits.
- **Traffic Light System**: Green/Amber/Red budget zones with request blocking at Red.
- **Structured Logging**: JSON-structured request logs via `slog`.

## Usage

1. Copy `tkngate.example.yaml` to `tkngate.yaml` and configure your API keys and budgets.
2. Run `go build` to build the `tkngate` executable.
3. Start the proxy with `./tkngate serve`.

### Commands

- `tkngate serve`: Starts the proxy daemon.
- `tkngate config validate`: Parses and validates the configuration file.
- `tkngate config show`: Prints resolved configuration (with secrets masked).
- `tkngate budget status`: Shows current spend vs. limits for each provider.
- `tkngate budget reset`: Manually resets the budget ledger.
