BINARY     := bacot
BIN_DIR    := bin
MODULE     := github.com/sphinxid/bacot
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS    := -X $(MODULE)/internal/version.Version=$(VERSION) \
              -X $(MODULE)/internal/version.Commit=$(COMMIT) \
              -X $(MODULE)/internal/version.BuildDate=$(BUILD_DATE)

.PHONY: build test install release clean lint tidy

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .

test:
	CGO_ENABLED=0 go test ./... -v -timeout 60s

test-race:
	go test ./... -v -race -timeout 60s

install:
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" .

release:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-amd64   .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-darwin-amd64  .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64  go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-darwin-arm64  .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64  go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-windows-amd64.exe .

clean:
	rm -rf $(BIN_DIR)

lint:
	golangci-lint run ./...

tidy:
	go mod tidy
