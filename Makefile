.PHONY: help build up down nuke logs run fix fmt lint vet test check coverage coverage-html

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

## --- Local stack ---

build: ## Build and start the local stack
	docker compose up -d --build

up: ## Start the local stack
	docker compose up -d

down: ## Stop the local stack
	docker compose down

nuke: ## Stop the local stack and destroy its volumes
	docker compose down -v

logs: ## Follow application logs
	docker compose logs -f app

run: ## Run the service locally, outside Docker
	go run ./cmd/relay serve

## --- Quality ---

fix:
	go fix ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

vet:
	go vet ./...

check: fix fmt vet lint ## Everything CI checks, locally

## --- Tests ---

test: ## Run tests
	go test ./...

coverage: ## Coverage with a per-function summary
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

coverage-html: coverage ## Open the coverage report in a browser
	go tool cover -html=coverage.out
