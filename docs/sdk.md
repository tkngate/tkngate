# Tkngate Polyglot SDKs

Tkngate provides lightweight, zero-dependency middleware SDKs for almost every major programming language. 

Because Tkngate is a zero-trust sidecar, these SDKs do not replace your existing LLM libraries (like the official OpenAI or Anthropic clients). Instead, they act as transparent interceptors that automatically route your traffic through the sidecar and inject the necessary security credentials (`Authorization: Bearer tkngate-sk-...`) and routing headers.

## Multi-Provider Routing

Tkngate seamlessly supports 8 distinct LLM providers:
**OpenAI, Anthropic, DeepSeek, Mistral, Kimi, Groq, Ollama, and Gemini**.

You can dynamically target any of these providers using the same official SDK (e.g., the standard OpenAI library) simply by changing the `provider` argument in our wrappers!


## Supported Languages

### Node.js / TypeScript
**Installation:**
```bash
npm install tkngate
```
**Usage:**
```javascript
const { wrapConfig, wrapAnthropicConfig } = require('tkngate');

// OpenAI (or DeepSeek/Groq via the OpenAI client)
const openai = new OpenAI(wrapConfig(
  {}, null, "http://localhost:7477/v1", { provider: "deepseek" }
));

// Anthropic
const anthropic = new Anthropic(wrapAnthropicConfig());
```

### Python
**Installation:**
```bash
pip install tkngate
```
**Usage:**
```python
from tkngate import wrap, wrap_anthropic

# OpenAI
client = wrap(OpenAI())

# Anthropic
client = wrap_anthropic(Anthropic())
```

### Go
**Installation:**
```bash
go get github.com/tkngate/tkngate/sdks/go/tkngate
```
**Usage:**
```go
import "github.com/tkngate/tkngate/sdks/go/tkngate"

// Wrap the standard HTTP client
client := tkngate.WrapClient(nil, "tkngate-sk-...", "openai", "session-123")
```

### Java
**Installation (Maven):**
```xml
<dependency>
    <groupId>com.tkngate</groupId>
    <artifactId>tkngate-sdk</artifactId>
    <version>1.0.0</version>
</dependency>
```
**Usage:**
```java
import com.tkngate.TkngateInterceptor;

OkHttpClient client = new OkHttpClient.Builder()
    .addInterceptor(new TkngateInterceptor("tkngate-sk-...", "openai", "session-123"))
    .build();
```

### Ruby
**Installation:**
```bash
gem install tkngate
```
**Usage:**
```ruby
require 'tkngate/faraday_middleware'

conn = Faraday.new do |f|
  f.request :tkngate
end
```

### C# / .NET
**Installation:**
```bash
dotnet add package Tkngate
```
**Usage:**
```csharp
using Tkngate;

var client = new HttpClient(new TkngateHandler());
```

### Rust
**Installation:**
```bash
cargo add tkngate
```
**Usage:**
```rust
use reqwest_middleware::ClientBuilder;
use tkngate::TkngateMiddleware;

let reqwest_client = reqwest::Client::new();
let client = ClientBuilder::new(reqwest_client)
    .with(TkngateMiddleware::new(None, None, None))
    .build();
```

### PHP
**Installation:**
```bash
composer require tkngate/tkngate
```
**Usage:**
```php
use Tkngate\TkngateMiddleware;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Client;

$stack = HandlerStack::create();
$stack->push(new TkngateMiddleware());
$client = new Client(['handler' => $stack]);
```
