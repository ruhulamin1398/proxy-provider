.PHONY: run build test lint tidy clean

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