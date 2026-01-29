.PHONY: build clean test lint install

ROOT := $(shell pwd -P)
GIT_COMMIT := $(shell git --work-tree ${ROOT} rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_TAG := $(shell git --work-tree ${ROOT} describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
BUILD_DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')

BINARY_NAME := n9e-mcp-server
LDFLAGS := -w -s \
	-X main.version=$(GIT_TAG) \
	-X main.commit=$(GIT_COMMIT) \
	-X main.date=$(BUILD_DATE)

all: build

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./cmd/n9e-mcp-server/

clean:
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

lint:
	golangci-lint run ./...

install: build
	cp $(BINARY_NAME) $(GOPATH)/bin/

version:
	@echo "Version: $(GIT_TAG)"
	@echo "Commit:  $(GIT_COMMIT)"
	@echo "Date:    $(BUILD_DATE)"
