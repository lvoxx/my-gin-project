# =============================================================================
# Employee Management System — Makefile
# =============================================================================

APP_NAME   := simple-employees-crud
BINARY_DIR := ./bin
BINARY     := $(BINARY_DIR)/$(APP_NAME)
CMD        := ./cmd/server
SWAGGER_OUT := ./docs

# Colour helpers
GREEN  := \033[0;32m
YELLOW := \033[0;33m
RESET  := \033[0m

.PHONY: help build run dev test lint fmt swagger seed-admin clean tidy

# ─── Help ─────────────────────────────────────────────────────────────────────
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(YELLOW)Usage:$(RESET)\n  make $(GREEN)<target>$(RESET)\n\nTargets:\n"} \
	     /^[a-zA-Z_-]+:.*?##/ { printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ─── Build ────────────────────────────────────────────────────────────────────
build: ## Compile the binary to ./bin/
	@mkdir -p $(BINARY_DIR)
	@echo "$(GREEN)Building $(APP_NAME)...$(RESET)"
	go build -ldflags="-s -w" -o $(BINARY) $(CMD)
	@echo "$(GREEN)Binary: $(BINARY)$(RESET)"

# ─── Run ──────────────────────────────────────────────────────────────────────
run: build ## Build then run the server
	$(BINARY)

dev: ## Run with hot-reload (requires air: go install github.com/air-verse/air@latest)
	@command -v air >/dev/null 2>&1 || { echo "Install air: go install github.com/air-verse/air@latest"; exit 1; }
	air -c .air.toml

# ─── Dependencies ─────────────────────────────────────────────────────────────
tidy: ## Run go mod tidy and verify
	go mod tidy
	go mod verify

# ─── Testing ──────────────────────────────────────────────────────────────────
test: ## Run all tests with race detector
	go test -race -count=1 ./...

test-cover: ## Run tests with HTML coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report: coverage.html$(RESET)"

# ─── Code quality ─────────────────────────────────────────────────────────────
lint: ## Run golangci-lint (requires: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1; }
	golangci-lint run ./...

fmt: ## Format all Go source files
	gofmt -w -s .
	goimports -w . 2>/dev/null || true

vet: ## Run go vet
	go vet ./...

# ─── Swagger docs ─────────────────────────────────────────────────────────────
swagger: ## Generate Swagger/OpenAPI docs (requires: go install github.com/swaggo/swag/cmd/swag@latest)
	@command -v swag >/dev/null 2>&1 || { echo "Install: go install github.com/swaggo/swag/cmd/swag@latest"; exit 1; }
	swag init \
		--generalInfo $(CMD)/main.go \
		--output $(SWAGGER_OUT) \
		--parseDependency \
		--parseInternal
	@echo "$(GREEN)Swagger docs written to $(SWAGGER_OUT)$(RESET)"

# ─── Database helpers ─────────────────────────────────────────────────────────
seed-admin: ## Generate a bcrypt hash for the default admin password (Admin@123)
	@echo "Generating bcrypt hash for 'Admin@123'..."
	@go run -v ./scripts/hash_password/main.go Admin@123 2>/dev/null || \
	  python3 -c "import bcrypt; print(bcrypt.hashpw(b'Admin@123', bcrypt.gensalt(12)).decode())" || \
	  echo "Install python3-bcrypt or run the Go script manually."

# ─── Cleanup ─────────────────────────────────────────────────────────────────
clean: ## Remove build artefacts and log files
	rm -rf $(BINARY_DIR) coverage.out coverage.html
	@echo "$(GREEN)Cleaned$(RESET)"
