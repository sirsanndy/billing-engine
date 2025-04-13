GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run
LINTCMD=golangci-lint
SWAGCMD=swag

# Project variables
BINARY_NAME=billing-engine
CMD_PATH=./cmd/
BIN_DIR=./bin

HAS_LINTER := $(shell command -v $(LINTCMD) 2> /dev/null)
HAS_SWAG := $(shell command -v $(SWAGCMD) 2> /dev/null)

.PHONY: all build run start clean lint swag help tidy deps

default: help

all: build

build: tidy
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_PATH)/main.go
	@echo "Build complete: $(BIN_DIR)/$(BINARY_NAME)"

run: tidy swag
	@echo "Running $(BINARY_NAME) using go run..."
	$(GORUN) $(CMD_PATH)/main.go

start: build
	@echo "Starting $(BINARY_NAME) from binary..."
	$(BIN_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -rf ./vendor
	@# rm -f coverage.* profile.* # Removed test artifacts

swag: 
ifndef HAS_SWAG
	@echo ">>> WARNING: swag CLI not found. Skipping Swagger generation."
	@echo ">>> Please install it: go install github.com/swaggo/swag/cmd/swag@latest"
	# @exit 1 # Optional: exit if swag is mandatory
else
	@echo "Generating Swagger docs..."
	$(SWAGCMD) init -g $(CMD_PATH)/main.go -o ./docs --parseDependency --parseInternal
	@echo "Swagger docs generated/updated in ./docs"
endif

tidy:
	@echo "Running go mod tidy..."
	$(GOMOD) tidy

deps:
	@echo "Running go mod download..."
	$(GOMOD) download

help: 
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort