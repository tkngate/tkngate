# Budget System & Traffic Lights

Tkngate's budget system prevents runaway API spend using a real-time **Traffic Light** model.

## Traffic Light Zones

| Zone     | Condition                       | Behaviour                                      |
|----------|----------------------------------|------------------------------------------------|
| 🟢 GREEN  | Spend < Amber threshold         | All requests pass through normally             |
| 🟡 AMBER  | Spend ≥ 70% of global limit     | Requests pass but warnings are logged          |
| 🔴 RED    | Spend ≥ 95% of global limit     | Non-essential requests blocked (HTTP 429)       |

## Configuration

```yaml
budget:
  global_limit_usd: 50.00       # Total monthly cap
  max_session_cost_usd: 5.00    # Per-session cap
  amber_threshold_pct: 70       # Warn at 70%
  red_threshold_pct: 95         # Block at 95%
  reset_interval: "monthly"
```

## Session-Level Budgets

Each unique `X-Session-ID` header gets its own budget envelope. When a session exceeds its `max_session_cost_usd`, only that session is throttled — other sessions continue operating normally.

## Cost Estimation

Token counts are estimated using a tiktoken-compatible counter. Cost is calculated per-provider using the pricing table in `internal/tokenizer/counter.go`, which includes rates for OpenAI, Anthropic, DeepSeek, Kimi, and Groq.
