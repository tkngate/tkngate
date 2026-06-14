// Tkngate + OpenAI Node.js SDK Example
//
// This script shows how to route any existing OpenAI SDK code
// through Tkngate by changing exactly ONE line: the baseURL.
//
// Prerequisites:
//   npm install openai
//
// Usage:
//   1. Start Tkngate:   tkngate serve
//   2. Run this script:  node agent.js

import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:7477/openai/v1", // <-- Point to Tkngate instead of api.openai.com
  apiKey: "tkngate-sk-YOUR_VIRTUAL_KEY_HERE", // <-- Use your Tkngate Virtual Key
});

async function main() {
  const response = await client.chat.completions.create({
    model: "gpt-4o",
    messages: [
      { role: "system", content: "You are a helpful assistant." },
      { role: "user", content: "Explain what a reverse proxy is in one sentence." },
    ],
  });

  console.log("Model:", response.model);
  console.log("Reply:", response.choices[0].message.content);
  console.log("Tokens:", response.usage.total_tokens);
}

main();
