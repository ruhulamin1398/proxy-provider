# Proxy Provider

A free AI proxy server that gets you past free-tier limits.

Most AI providers give a free tier per API key. But some (like OpenCode)
also rate-limit your IP/PC — so when one key hits its limit, your other
fresh unused keys get blocked too, because they all share your IP.

This proxy fixes that. Requests are sent through this server (a different
IP), so every key gets a fresh start. Create another free key, and use
its full quota through the proxy — no more "limit reached" errors.

Two ways to use it:

1. **Direct API** — send your provider URL, API key, and model, and it
   forwards the request.
2. **OpenAI-compatible endpoints** — it acts like an OpenAI server, so
   Hermes or any OpenAI SDK can use it as a drop-in provider.

Deployed at **https://proxy-provider.onrender.com**

## Quick use

### 1. Direct api

```http
POST /proxy HTTP/1.1
Host: proxy-provider.onrender.com
Content-Type: application/json
Authorization: Bearer sk-...

{
  "base_url": "https://api.openai.com/v1",
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

The API key is sent via the `Authorization: Bearer <key>` header. The
`api_key` body field is still accepted for backward compatibility, but
the header takes precedence.

Response:

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "success": true,
  "data": {
    "upstream": "https://api.openai.com/v1",
    "content": "Hello! How can I help you today?",
    "finish_reason": "stop",
    "model": "gpt-4",
    "prompt_tokens": 9,
    "output_tokens": 12,
    "total_tokens": 21
  }
}
```

### 2. Ai provider at hermes

```yaml
# ~/.hermes/config.yaml
custom_providers:
  - name: proxy
    api_base: https://proxy-provider.onrender.com/v1
    api_key: your-api-key
    models:
      - deepseek-v4-flash-free
      - mimo-v2.5-free
      - ling-3.0-flash-free
```

Then select it in any channel:

```
/model set proxy/deepseek-v4-flash-free
```

## Prerequisites

- Go 1.25+ ([install](https://go.dev/dl/))

## Quick Start

```bash
# Clone and enter the project
cd proxy-provider

# Create a .env file (optional, defaults are shown below)
cat > .env << 'EOF'
PORT=8080
LOG_LEVEL=info
EOF

# Install dependencies and run
make run
```

The server starts on **http://localhost:8080**.

## Available Commands

| Command            | Like `pnpm ...` | Description                             |
|--------------------|-----------------|-----------------------------------------|
| `make dev`         | `pnpm dev`      | Run with live reload (via `air`)        |
| `make run`         | `pnpm dev`      | Run directly with `go run`              |
| `make build`       | `pnpm build`    | Compile to a binary in `bin/`           |
| `make serve`       | `pnpm serve`    | Run the pre-built binary                |
| `make start`       | —               | Build + serve in one step               |
| `make test`        | `pnpm test`     | Run all tests                           |
| `make clean`       | —               | Remove build artifacts                  |
| `make docker-build`| —               | Build Docker image                      |

> 💡 For **live reload** (`make dev`), install [air](https://github.com/air-verse/air):
> ```bash
> go install github.com/air-verse/air@latest
> ```

## Routes

| Method | Path              | Description                                       |
|--------|-------------------|---------------------------------------------------|
| GET    | `/health`         | Health check                                      |
| GET    | `/logs`           | View request logs (newest first)                  |
| GET    | `/logs/clear`     | Clear all logs                                    |
| POST   | `/proxy`          | Proxy to any OpenAI-compatible provider           |
| POST   | `/v1/chat/completions` | OpenAI-compatible chat completions           |
| GET    | `/v1/models`      | List available models                             |
| POST   | `/chat/completions`    | OpenAI-compatible (without `/v1` prefix)     |
| GET    | `/models`         | List models (without `/v1` prefix)                |

## Environment Variables

| Variable   | Default   | Description     |
|------------|-----------|-----------------|
| `PORT`     | `8080`    | Server port     |
| `LOG_LEVEL`| `info`    | Log level       |

## Usage

### Chat Completions (OpenAI-compatible)

```bash
POST /v1/chat/completions
Content-Type: application/json
Authorization: Bearer ***

{
  "model": "deepseek-v4-flash-free",
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

The endpoint also supports **SSE streaming** — add `"stream": true` to receive the response token-by-token.

### Proxy (any OpenAI-compatible provider)

```bash
POST /proxy
Content-Type: application/json

{
  "base_url": "https://api.openai.com/v1",
  "api_key": "sk-...",
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

## Logging

All incoming requests and downstream API calls are logged to `log.txt`.

- **Request logs**: `[timestamp] IP METHOD PATH > STATUS_CODE (duration)`
- **Downstream logs**: `[DOWNSTREAM] timestamp | model=... | url=... | status=... | tokens=... | req=... | resp=...`

View logs at `GET /logs` or by reading `log.txt` directly.

## Docker

### Build the image

```bash
# Using Make
make docker-build

# Or manually
docker build -t proxy-provider:latest .
```

### Run the container

```bash
docker run -d \
  --name proxy-provider \
  -p 8080:8080 \
  -e PORT=8080 \
  -e LOG_LEVEL=info \
  proxy-provider:latest
```

### Publish to Docker Hub

```bash
# 1. Tag the image with your Docker Hub username
docker tag proxy-provider:latest your-dockerhub-user/proxy-provider:latest

# 2. Log in to Docker Hub
docker login

# 3. Push the image
docker push your-dockerhub-user/proxy-provider:latest

# Or use the Make target (set IMAGE_NAME first)
make docker-build docker-push IMAGE_NAME=your-dockerhub-user/proxy-provider TAG=v1.0
```

### Deploy to Render

This project includes a [`render.yaml`](./render.yaml) for one-click deployment on [Render](https://render.com). Push the Docker image to a registry, then create a new Web Service on Render pointing to your image.
