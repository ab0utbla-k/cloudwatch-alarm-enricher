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
# Build Dependencies

## Binaries
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
GOLANGCI_VERSION := v2.4.0

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary
$(GOLANGCI_LINT): $(LOCALBIN)
	@test -s $(LOCALBIN)/golangci-lint && $(LOCALBIN)/golangci-lint version --format short | grep -q $(GOLANGCI_VERSION) || \
	(echo "Installing golangci-lint $(GOLANGCI_VERSION)..." && \
	 curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_VERSION))

# ====================================================================================
# Go Development

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
# Lambda Deployment

LAMBDA_FUNCTION_NAME ?= cloudwatch-alarm-enricher
BUILD_ARGS ?= CGO_ENABLED=0

.PHONY: build
build: ## Build Lambda binary
	@$(INFO) Building Lambda binary
	$(BUILD_ARGS) GOOS=linux GOARCH=amd64 go build -o bootstrap cmd/lambda/main.go
	@$(OK) Built Lambda binary

.PHONY: zip
zip: build ## Create Lambda deployment package
	@$(INFO) Creating deployment package
	@zip -q lambda.zip bootstrap
	@$(OK) Created lambda.zip

.PHONY: deploy
deploy: zip ## Deploy Lambda function to AWS
	@$(INFO) Deploying Lambda function
	@aws lambda update-function-code --function-name $(LAMBDA_FUNCTION_NAME) --zip-file fileb://lambda.zip
	@$(OK) Deployed Lambda function

.PHONY: clean
clean: ## Remove build artifacts
	@rm -f bootstrap lambda.zip
	@$(OK) Cleaned build artifacts

# ====================================================================================
# Build Artifacts
DOCKERFILE ?= Dockerfile
DOCKER ?= docker
DOCKER_BUILD_ARGS ?=

# Image registry for build/push image targets
export IMAGE_REGISTRY ?= ghcr.io
export IMAGE_REPO     ?= ab0utbla-k/cloudwatch-alarm-enricher
export IMAGE_NAME ?= $(IMAGE_REGISTRY)/$(IMAGE_REPO)

.PHONY: docker.image
docker.image:  ## Emit IMAGE_NAME:IMAGE_TAG
	@echo $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: docker.imagename
docker.imagename:  ## Emit IMAGE_NAME
	@echo $(IMAGE_NAME)

.PHONY: docker.tag
docker.tag:  ## Emit IMAGE_TAG
	@echo $(IMAGE_TAG)

.PHONY: docker.build
docker.build: $(addprefix build-,$(ARCH)) ## Build the docker image
	@$(INFO) $(DOCKER) build
	echo $(DOCKER) build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	DOCKER_BUILDKIT=1 $(DOCKER) build -f $(DOCKERFILE) . $(DOCKER_BUILD_ARGS) -t $(IMAGE_NAME):$(IMAGE_TAG)
	@$(OK) $(DOCKER) build

.PHONY: docker.push
docker.push: ## Push the docker image to the registry
	@$(INFO) $(DOCKER) push
	@$(DOCKER) push $(IMAGE_NAME):$(IMAGE_TAG)
	@$(OK) $(DOCKER) push
