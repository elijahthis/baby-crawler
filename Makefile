# --- Configuration ---
BINARY_DIR=bin
CRAWLER_BINARY=crawler
PARSER_BINARY=parser

# Default seed for local runs
SEED_URL="https://grpc.io/docs/"

# Redis/MinIO Defaults for local non-docker runs
REDIS_ADDR="localhost:6379"
S3_ENDPOINT="http://localhost:9000"

# --- Main Targets ---

.PHONY: all build run-crawler run-parser docker-up docker-down clean test help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

all: build ## Build both services

build: ## Compile binaries to ./bin
	@echo "Building binaries..."
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_DIR)/$(CRAWLER_BINARY) ./cmd/crawler
	go build -o $(BINARY_DIR)/$(PARSER_BINARY) ./cmd/parser
	@echo "Build complete."

# --- Local Development (No Docker) ---

run-crawler: build ## Run the crawler locally
	@echo "Starting Crawler (Seed: $(SEED_URL))..."
	./$(BINARY_DIR)/$(CRAWLER_BINARY) \
		--seed=$(SEED_URL) \
		--redis-addr=$(REDIS_ADDR) \
		--s3-endpoint=$(S3_ENDPOINT)

run-parser: build ## Run the parser locally
	@echo "Starting Parser..."
	./$(BINARY_DIR)/$(PARSER_BINARY) \
		--redis-addr=$(REDIS_ADDR) \
		--s3-endpoint=$(S3_ENDPOINT)

# --- Docker Operations ---

up: ## Start the full stack (Redis + MinIO + Services)
	docker-compose up --build

down: ## Stop all containers
	docker-compose down

scale: ## Scale the parser to 5 instances
	docker-compose up -d --scale parser=5 --no-recreate

logs: ## Tail logs for all services
	docker-compose logs -f

# --- Maintenance ---

test: ## Run unit tests
	go test -v ./...

clean: ## Remove binaries and temp files
	rm -rf $(BINARY_DIR)
	go clean

deps: ## Tidy and download dependencies
	go mod tidy
	go mod download