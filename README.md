# Proxy Provider

A minimal Go server that proxies requests to OpenAI-compatible providers.

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

# Download dependencies and build
go mod tidy
go build -o app ./cmd/server

# Run the server
./app
```

The server starts on **http://localhost:8080**.

### Using `run.sh`

Alternatively, use the provided script:

```bash
chmod +x run.sh
./run.sh
```

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
Authorization: Bearer your-api-key

{
  "model": "deepseek-v4-flash-free",
  "messages": [
    {"role": "user", "content": "Hello"}
  ]
}
```

The endpoint also supports **SSE streaming** — add `"stream": true` to receive the response token-by-token.
```

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
  ],
  "temperature": 0.7,
  "max_tokens": 100
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