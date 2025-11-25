.DEFAULT_GOAL := help

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
# Variables - Local Lambda Deployment

LAMBDA_FUNCTION_NAME ?= cloudwatch-alarm-enricher
LAMBDA_BUILD_ARGS ?= CGO_ENABLED=0
LAMBDA_BINARY ?= bootstrap
LAMBDA_ZIP ?= lambda.zip

# ====================================================================================
# Variables - Docker/Container

DOCKERFILE ?= Dockerfile
DOCKER ?= docker
DOCKER_BUILD_ARGS ?=

# Image registry for build/push image targets
export IMAGE_REGISTRY ?= ghcr.io
export IMAGE_REPO     ?= ab0utbla-k/cloudwatch-alarm-enricher
export IMAGE_NAME ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)
export IMAGE_TAG ?= latest

# ====================================================================================
# Variables - Build Tools

LOCALBIN ?= $(shell pwd)/bin
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOLANGCI_VERSION := v2.6.2

# ====================================================================================
# Help

.PHONY: help
help: ## Display this help message
	@echo "$(BLUE)Available targets:$(CNone)"
	@echo ""
	@echo "$(YELLOW)Development:$(CNone)"
	@grep -E '^(test|lint|fmt|tidy):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(CNone) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Local Lambda Deployment:$(CNone)"
	@grep -E '^lambda\.[^:]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(CNone) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Docker/Container:$(CNone)"
	@grep -E '^docker\.[^:]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(CNone) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Tools:$(CNone)"
	@grep -E '^(golangci-lint):.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(CNone) %s\n", $$1, $$2}'

# ====================================================================================
# Development Targets

.PHONY: test
test: ## Run tests
	@$(INFO) Running tests
	@go test -v ./...
	@$(OK) Tests passed

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	@$(INFO) Running linter
	@if ! $(GOLANGCI_LINT) run; then \
		printf "\033[0;33mgolangci-lint failed: some checks can be fixed with \`\033[0;32mmake fmt\033[0m\033[0;33m\`\033[0m\n"; \
		exit 1; \
	fi
	@$(OK) Finished linting

.PHONY: fmt
fmt: $(GOLANGCI_LINT) ## Format code and fix issues
	@$(INFO) Formatting code
	@go mod tidy
	@$(GOLANGCI_LINT) fmt
	@$(GOLANGCI_LINT) run --fix
	@$(OK) Code formatted

.PHONY: tidy
tidy: ## Tidy go modules
	@$(INFO) Tidying go modules
	@go mod tidy
	@$(OK) Modules tidied

# ====================================================================================
# Local Lambda Deployment Targets

.PHONY: lambda.build
lambda.build: ## Build Lambda binary for local deployment
	@$(INFO) Building Lambda binary
	$(LAMBDA_BUILD_ARGS) GOOS=linux GOARCH=amd64 go build -o $(LAMBDA_BINARY) cmd/lambda/main.go
	@$(OK) Built Lambda binary

.PHONY: lambda.zip
lambda.zip: lambda.build ## Create Lambda deployment package
	@$(INFO) Creating deployment package
	@zip -q $(LAMBDA_ZIP) $(LAMBDA_BINARY)
	@$(OK) Created $(LAMBDA_ZIP)

.PHONY: lambda.deploy
lambda.deploy: lambda.zip ## Deploy Lambda function to AWS (local development)
	@$(INFO) Deploying Lambda function
	@aws lambda update-function-code --function-name $(LAMBDA_FUNCTION_NAME) --zip-file fileb://$(LAMBDA_ZIP)
	@$(OK) Deployed Lambda function

.PHONY: lambda.clean
lambda.clean: ## Remove Lambda build artifacts
	@$(INFO) Cleaning Lambda build artifacts
	@rm -f $(LAMBDA_BINARY) $(LAMBDA_ZIP)
	@$(OK) Cleaned build artifacts

# Convenience aliases for backwards compatibility
.PHONY: build
build: lambda.build

.PHONY: deploy
deploy: lambda.deploy

.PHONY: clean
clean: lambda.clean

# ====================================================================================
# Docker/Container Targets

.PHONY: docker.build
docker.build: ## Build the docker image
	@$(INFO) $(DOCKER) build
	@echo $(DOCKER) build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	@DOCKER_BUILDKIT=1 $(DOCKER) build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	@$(OK) $(DOCKER) build

.PHONY: docker.push
docker.push: ## Push the docker image to the registry
	@$(INFO) $(DOCKER) push
	@$(DOCKER) push $(IMAGE_NAME):$(IMAGE_TAG)
	@$(OK) $(DOCKER) push

.PHONY: docker.image
docker.image: ## Display full image name with tag
	@echo $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: docker.imagename
docker.imagename: ## Display image name without tag
	@echo $(IMAGE_NAME)

.PHONY: docker.tag
docker.tag: ## Display image tag
	@echo $(IMAGE_TAG)

# ====================================================================================
# Build Tools

$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary
$(GOLANGCI_LINT): $(LOCALBIN)
	@test -s $(LOCALBIN)/golangci-lint && $(LOCALBIN)/golangci-lint version --format short | grep -q $(GOLANGCI_VERSION) || \
	(echo "Installing golangci-lint $(GOLANGCI_VERSION)..." && \
	 curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_VERSION))
