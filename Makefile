.SHELL := /usr/bin/env bash

# Module and binary information 
PROXY_MAIN_PKG := cmd/whisper-proxy
PROXY_MAIN_PATH := $(PROXY_MAIN_PKG)/main.go
PROXY_BINARY := bin/whisper-proxy
MODULE_NAME := $(shell head -n 1 go.mod | cut -d' ' -f2)

# Can contain additional arguments to pass to the go run command, e.g. make GOAL ARGS="--config config.json"
ARGS ?= 

# Build metadata
VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Check if the git working directory is dirty (has uncommitted changes)
GIT_STATUS := $(shell git status --porcelain 2>/dev/null)
ifneq ($(GIT_STATUS),)
    COMMIT := $(COMMIT)-dirty
endif

# Go compiler flags (-ldflags injects the metadata into your binary)
LDFLAGS     := -ldflags "-X $(MODULE_NAME)/$(PROXY_MAIN_PKG)/cmd.Version=$(VERSION) \
                         -X $(MODULE_NAME)/$(PROXY_MAIN_PKG)/cmd.Commit=$(COMMIT) \
                         -X $(MODULE_NAME)/$(PROXY_MAIN_PKG)/cmd.BuildTime=$(BUILD_TIME)"

.PHONY: run
run:
	@go run $(LDFLAGS) $(PROXY_MAIN_PATH) $(ARGS)

.PHONY: build
build:
	@mkdir -p $(dir $(PROXY_BINARY))
	@go build $(LDFLAGS) -o $(PROXY_BINARY) $(PROXY_MAIN_PATH) $(ARGS)

.PHONY: tidy
tidy:
	@go mod tidy

.PHONY: fmt
fmt:
	@go fmt ./... 
	go run golang.org/x/tools/cmd/goimports@latest -w .
	
.PHONY: clean
clean:
	@rm -rf $(dir $(PROXY_BINARY))

