# Tkngate + OpenAI Python SDK Example
#
# This script shows how to route any existing OpenAI SDK code
# through Tkngate by changing exactly ONE line: the base_url.
#
# Prerequisites:
#   pip install openai
#
# Usage:
#   1. Start Tkngate:   tkngate serve
#   2. Run this script:  python agent.py

from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:7477/openai/v1",  # <-- Point to Tkngate instead of api.openai.com
    api_key="tkngate-sk-YOUR_VIRTUAL_KEY_HERE",   # <-- Use your Tkngate Virtual Key
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {"role": "system", "content": "You are a helpful assistant."},
        {"role": "user", "content": "Explain what a reverse proxy is in one sentence."},
    ],
)

print("Model:", response.model)
print("Reply:", response.choices[0].message.content)
print("Tokens:", response.usage.total_tokens)
