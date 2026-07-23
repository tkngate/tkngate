# tkngate

The official Node.js SDK for [Tkngate](https://github.com/tkngate/tkngate) - The zero-trust P2P gateway for LLMs.

Tkngate allows you to route your LLM requests through a local or distributed proxy to enforce budgets, RBAC, WAF rules, and semantic caching, all while maintaining perfect compatibility with official SDKs.

## Installation

```bash
npm install tkngate openai
```

## Usage

Simply wrap your OpenAI client with `wrapConfig`:

```javascript
const { OpenAI } = require('openai');
const { wrapConfig } = require('tkngate');

// Initialize the OpenAI client wrapped with Tkngate routing
const client = new OpenAI(wrapConfig(
  { defaultHeaders: {} },           // Base config
  "tkngate-sk-YOUR-VIRTUAL-KEY",    // Your Tkngate Virtual Key
  "http://localhost:7477/openai/v1",// The URL to your Tkngate Proxy
  {
    provider: "openai",             // Target provider (openai, anthropic, gemini, etc.)
    sessionId: "my-session-123"     // Optional: track requests by session ID
  }
));

// Use the client exactly as you normally would!
async function main() {
  const response = await client.chat.completions.create({
    model: "gpt-4",
    messages: [{ role: "user", content: "Hello from Tkngate!" }],
  });
  console.log(response.choices[0].message.content);
}

main();
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
