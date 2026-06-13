# Shadow Mode (Traffic Mirroring)

Shadow Mode allows enterprises to silently evaluate alternative AI providers using real production traffic — with zero risk and zero latency impact.

## Use Case

Your company spends $50,000/month on `gpt-6o`. You want to test whether `deepseek-chat-v3` can produce equivalent results at 1/10th the cost. But you can't risk breaking your production agents.

With Shadow Mode enabled, Tkngate will:
1. Process the primary request normally through `gpt-6o`.
2. Asynchronously clone a fraction of that traffic to `deepseek-chat-v3` in the background.
3. Log the shadow response's latency and status for comparison.

The shadow request runs in a separate goroutine and **never** blocks or delays the primary response.

## Configuration

```yaml
shadow:
  enabled: true
  target_provider: "deepseek"
  target_model: "deepseek-chat-v3"
  traffic_fraction: 0.1       # Mirror 10% of traffic
```

## Safety

- Shadow Mode only activates on `/chat/completions` endpoints.
- The shadow goroutine includes panic recovery to prevent crashes.
- Shadow requests use a 30-second timeout to prevent resource leaks.
- Shadow traffic does not count towards your budget ledger.
