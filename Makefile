.PHONY: run dev build serve start test lint tidy clean docker-build docker-push

# Default image name — override with IMAGE_NAME=<your-dockerhub-user>/proxy-provider
IMAGE_NAME ?= proxy-provider
TAG      ?= latest

BIN_DIR  ?= bin
BINARY   ?= $(BIN_DIR)/server

## ─── Development ──────────────────────────────────────────

# dev: run with live reload (requires `air` — install: go install github.com/air-verse/air@latest)
dev:
	@command -v air >/dev/null 2>&1 || { \
		echo "air not found. Install: go install github.com/air-verse/air@latest"; \
		echo "Falling back to 'go run'..."; \
		go run ./cmd/server/main.go; \
		exit 1; \
	}
	air

# run: run the server directly with go run
run:
	@go run ./cmd/server/main.go

# build: compile the binary
build:
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 go build -o $(BINARY) ./cmd/server

# serve: run the pre-built binary (run `make build` first)
serve: $(BINARY)
	@echo "Starting server on http://localhost:$${PORT:-8080}"
	./$(BINARY)

# start: build + serve in one step
start: build serve

# test: run all tests
test:
	@go test -v -race -count=1 ./...

# lint: run linter
lint:
	@golangci-lint run ./...

# tidy: clean up dependencies
tidy:
	@go mod tidy

# clean: remove build artifacts
clean:
	@rm -rf $(BIN_DIR)/

$(BINARY):
	$(MAKE) build

## ─── Docker ────────────────────────────────────────────────

docker-build:
	docker build -t $(IMAGE_NAME):$(TAG) .

docker-push:
	docker push $(IMAGE_NAME):$(TAG)