# Tkngate Polyglot SDKs

Tkngate provides lightweight, zero-dependency middleware SDKs for almost every major programming language. 

Because Tkngate is a zero-trust sidecar, these SDKs do not replace your existing LLM libraries (like the official OpenAI or Anthropic clients). Instead, they act as transparent interceptors that automatically route your traffic through the sidecar and inject the necessary security credentials (`Authorization: Bearer tkngate-sk-...`) and routing headers.

## Supported Languages

### Node.js / TypeScript
```javascript
const { wrapConfig, wrapAnthropicConfig } = require('tkngate');

// OpenAI
const openai = new OpenAI(wrapConfig());

// Anthropic
const anthropic = new Anthropic(wrapAnthropicConfig());
```

### Python
```python
from tkngate import wrap, wrap_anthropic

# OpenAI
client = wrap(OpenAI())

# Anthropic
client = wrap_anthropic(Anthropic())
```

### Go
```go
import "tkngate"

// Wrap the standard HTTP client
client := tkngate.WrapClient(nil, "tkngate-sk-...", "openai", "session-123")
```

### Java
```java
import com.tkngate.TkngateInterceptor;

OkHttpClient client = new OkHttpClient.Builder()
    .addInterceptor(new TkngateInterceptor("tkngate-sk-...", "openai", "session-123"))
    .build();
```

### Ruby
```ruby
require 'tkngate/faraday_middleware'

conn = Faraday.new do |f|
  f.request :tkngate
end
```

### C# / .NET
```csharp
using Tkngate;

var client = new HttpClient(new TkngateHandler());
```

### Rust
```rust
use reqwest_middleware::ClientBuilder;
use tkngate::TkngateMiddleware;

let reqwest_client = reqwest::Client::new();
let client = ClientBuilder::new(reqwest_client)
    .with(TkngateMiddleware::new(None, None, None))
    .build();
```

### PHP
```php
use Tkngate\TkngateMiddleware;
use GuzzleHttp\HandlerStack;
use GuzzleHttp\Client;

$stack = HandlerStack::create();
$stack->push(new TkngateMiddleware());
$client = new Client(['handler' => $stack]);
```
