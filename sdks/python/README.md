# tkngate

The official Python SDK for [Tkngate](https://github.com/tkngate/tkngate) - The zero-trust P2P gateway for LLMs.

Tkngate allows you to route your LLM requests through a local or distributed proxy to enforce budgets, RBAC, WAF rules, and semantic caching, all while maintaining perfect compatibility with official SDKs.

## Installation

```bash
pip install tkngate openai
```

## Usage

Simply wrap your OpenAI client with the `wrap` function:

```python
from openai import OpenAI
from tkngate import wrap

# Initialize the OpenAI client wrapped with Tkngate routing
client = wrap(
    client=OpenAI(),
    virtual_key="tkngate-sk-YOUR-VIRTUAL-KEY",     # Your Tkngate Virtual Key
    proxy_url="http://localhost:7477/openai/v1",   # The URL to your Tkngate Proxy
    provider="openai",                             # Target provider
    session_id="my-session-123"                    # Optional: track requests by session ID
)

# Use the client exactly as you normally would!
response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Hello from Tkngate!"}]
)

print(response.choices[0].message.content)
```

## Supported Providers
Tkngate seamlessly supports routing for:
- OpenAI
- Anthropic
- Google Gemini
- DeepSeek
- Mistral
- Groq
- Kimi
- Ollama

## License
Apache 2.0
