# Strict Rate Limiting (Token Bucket)

Tkngate implements a pre-emptive **Token Bucket** rate limiter to protect your upstream API providers from autonomous agent "burst loops" or runaway scripts.

## Why is it needed?

Autonomous agents can sometimes get stuck in loops, making rapid-fire requests to the API. Without protection, this can quickly:
- Exhaust your budget.
- Cause your API keys to be permanently banned or temporarily blocked by the AI provider due to exceeding requests per minute (RPM) limits.

## How it works

Tkngate uses a highly-performant, in-memory token bucket implementation (`golang.org/x/time/rate`). It intercepts traffic before any payload rewriting or processing occurs. If the rate limit is exceeded, Tkngate short-circuits the request and instantly returns an HTTP `429 Too Many Requests` error.

The rate limiting is tracked independently per isolated session or Virtual Key.

## Configuration

Rate limiting is configured globally in your `tkngate.yaml` file:

```yaml
rate_limit:
  enabled: true
  requests_per_minute: 60
  burst: 10
```

- `enabled`: Toggles the rate limiter on or off.
- `requests_per_minute` (RPM): The steady-state rate at which new tokens are added to the bucket.
- `burst`: The maximum number of requests allowed in a sudden burst before the RPM limit is strictly enforced.
