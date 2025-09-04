BREW_PREFIX := $(shell brew --prefix)
BIN ?= $(BREW_PREFIX)/bin

## Tool Binaries
GOLANGCI_LINT ?= $(BIN)/golangci-lint

# ====================================================================================
# Colors

BLUE         := $(shell printf "\033[34m")
YELLOW       := $(shell printf "\033[33m")
RED          := $(shell printf "\033[31m")
GREEN        := $(shell printf "\033[32m")
CNone        := $(shell printf "\033[0m")

# ====================================================================================
# Logger

TIME_LONG	= `date +%Y-%m-%d' '%H:%M:%S`
TIME_SHORT	= `date +%H:%M:%S`
TIME		= $(TIME_SHORT)

INFO	= echo ${TIME} ${BLUE}[ .. ]${CNone}
WARN	= echo ${TIME} ${YELLOW}[WARN]${CNone}
ERR		= echo ${TIME} ${RED}[FAIL]${CNone}
OK		= echo ${TIME} ${GREEN}[ OK ]${CNone}
FAIL	= (echo ${TIME} ${RED}[FAIL]${CNone} && false)

# ====================================================================================
# Golang
.PHONY: lint
lint: ## Run golangci-lint
	@if ! $(GOLANGCI_LINT) run; then \
		printf "\033[0;33mgolangci-lint failed: some checks can be fixed with \`\033[0;32mmake fmt\033[0m\033[0;33m\`\033[0m\n"; \
		exit 1; \
	fi
	@$(OK) Finished linting

.PHONY: fmt
fmt: ## Ensure consistent code style
	@go mod tidy
	@go fmt ./...
	@$(GOLANGCI_LINT) run --fix
	@$(OK) Ensured consistent code style
