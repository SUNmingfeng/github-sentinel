.PHONY: help run monitor build clean

help:
	@echo "Available commands:"
	@echo "  make run      - Run the interactive GitHub Sentinel"
	@echo "  make monitor  - Run one-time monitor for langchain"
	@echo "  make build    - Build all binaries"
	@echo "  make clean    - Clean build artifacts"

run:
	@echo "Starting GitHub Sentinel interactive console..."
	@export GITHUB_TOKEN=$(GITHUB_TOKEN); go run cmd/server/main.go

monitor:
	@echo "Running one-time monitor for langchain-ai/langchain..."
	@export GITHUB_TOKEN=$(GITHUB_TOKEN); go run cmd/monitor/main.go

build:
	@echo "Building GitHub Sentinel..."
	@mkdir -p bin
	@go build -o bin/github-sentinel cmd/server/main.go
	@go build -o bin/monitor cmd/monitor/main.go

clean:
	@echo "Cleaning..."
	@rm -rf bin/ data/ *.md