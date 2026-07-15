# Koris Makefile
# Build automation for frontend and backend.
# Run `make help` (default) to list available targets.

PANEL_BIN   := koris
WEB_DIR     := web
GO_LDFLAGS  := -w -s

.DEFAULT_GOAL := help

# ─── Help ─────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nKoris make targets:\n\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 } /^# ───/ { printf "\n" }' $(MAKEFILE_LIST)

# ─── Frontend ─────────────────────────────────────────────────────────

.PHONY: frontend
frontend: ## Build all frontends (admin, portal, landing) with a frozen lockfile
	cd $(WEB_DIR) && pnpm install --frozen-lockfile
	cd $(WEB_DIR) && pnpm --filter admin build
	cd $(WEB_DIR) && pnpm --filter portal build
	cd $(WEB_DIR) && pnpm --filter landing build

.PHONY: frontend-dev
frontend-dev: ## Build all frontends without a frozen lockfile (local dev)
	cd $(WEB_DIR) && pnpm install
	cd $(WEB_DIR) && pnpm --filter admin build
	cd $(WEB_DIR) && pnpm --filter portal build
	cd $(WEB_DIR) && pnpm --filter landing build

.PHONY: clean-frontend
clean-frontend: ## Remove built frontend assets and node_modules
	rm -rf $(WEB_DIR)/admin/www
	rm -rf $(WEB_DIR)/portal/www
	rm -rf $(WEB_DIR)/landing/www
	rm -rf $(WEB_DIR)/node_modules

# ─── Backend ──────────────────────────────────────────────────────────

.PHONY: backend
backend: ## Build the Go panel binary (full edition)
	CGO_ENABLED=0 go build -ldflags="$(GO_LDFLAGS)" -o $(PANEL_BIN) ./cmd/panel

.PHONY: backend-lite
backend-lite: ## Build the Go panel binary (lite edition)
	CGO_ENABLED=0 go build -tags lite -ldflags="$(GO_LDFLAGS)" -o $(PANEL_BIN) ./cmd/panel

# ─── Combined ─────────────────────────────────────────────────────────

.PHONY: build
build: frontend backend ## Build frontends + backend (full)

.PHONY: build-lite
build-lite: frontend backend-lite ## Build frontends + backend (lite)

# ─── Proto ──────────────────────────────────────────────────────────

.PHONY: proto
proto: ## Regenerate internal/knodepb from ../knode/proto
	@command -v buf >/dev/null 2>&1 || { echo "buf not installed: go install github.com/bufbuild/buf/cmd/buf@latest"; exit 1; }
	buf generate ../knode/proto
	@echo "regenerated internal/knodepb from ../knode/proto"

# ─── Quality ──────────────────────────────────────────────────────────

.PHONY: vet
vet: ## Run go vet on all packages
	go vet ./...

.PHONY: test
test: ## Run Go unit tests
	go test ./...

.PHONY: test-frontend
test-frontend: ## Run frontend unit tests
	cd $(WEB_DIR) && pnpm test

.PHONY: check
check: vet test ## Run vet + tests (CI-style quick gate)

# ─── Clean ────────────────────────────────────────────────────────────

.PHONY: clean
clean: clean-frontend ## Remove all build artifacts
	rm -f $(PANEL_BIN)
