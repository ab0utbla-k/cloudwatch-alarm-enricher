# ====================================================================================
# Build Dependencies

## Binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

GOLANGCI_VERSION := v2.4.0

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	@test -s $(LOCALBIN)/golangci-lint && $(LOCALBIN)/golangci-lint version --format short | grep -q $(GOLANGCI_VERSION) || \
	(echo "Installing golangci-lint $(GOLANGCI_VERSION)..." && \
	 curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_VERSION))

.PHONY: clean-bin
clean-bin: ## Remove all downloaded tools
	@rm -rf $(LOCALBIN)
	@$(OK) Removed local binaries

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
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	@if ! $(GOLANGCI_LINT) run; then \
		printf "\033[0;33mgolangci-lint failed: some checks can be fixed with \`\033[0;32mmake fmt\033[0m\033[0;33m\`\033[0m\n"; \
		exit 1; \
	fi
	@$(OK) Finished linting

.PHONY: fmt
fmt: $(GOLANGCI_LINT) ## Ensure consistent code style
	@go mod tidy
	@$(GOLANGCI_LINT) fmt
	@$(GOLANGCI_LINT) run --fix
	@$(OK) Ensured consistent code style
