# Proxy Provider

A minimal Go server that proxies requests to OpenAI-compatible providers.

## Routes

| Method | Path      | Description                            |
|--------|-----------|----------------------------------------|
| GET    | `/health` | Health check                           |
| POST   | `/proxy`  | Proxy to any OpenAI-compatible provider |

## Usage

```bash
POST /proxy
Content-Type: application/json

{
  "base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "temperature": 0.7,
  "max_tokens": 100
}
```

Response contains `upstream`, `content`, `finish_reason`, `model`, and token counts.