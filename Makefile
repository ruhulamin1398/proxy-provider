.PHONY: run build test lint tidy clean docker-build docker-push

# Default image name — override with IMAGE_NAME=<your-dockerhub-user>/proxy-provider
IMAGE_NAME ?= proxy-provider
TAG      ?= latest

run:
	@go run ./cmd/server/main.go

build:
	@mkdir -p bin
	@CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	@go test -v -race -count=1 ./...

lint:
	@golangci-lint run ./...

tidy:
	@go mod tidy

clean:
	@rm -rf bin/

# ─── Docker ────────────────────────────────────────────────

docker-build:
	docker build -t $(IMAGE_NAME):$(TAG) .

docker-push:
	docker push $(IMAGE_NAME):$(TAG)