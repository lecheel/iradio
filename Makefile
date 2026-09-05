BINARY_NAME := iradio
DIST_DIR    := dist
SRC         := main.go
LDFLAGS     := -s -w

.PHONY: all build run clean deps linux linux-amd64 linux-arm64 macos darwin-amd64 darwin-arm64 build-all

all: build

## build: Build binary for current host platform
build:
	@mkdir -p $(DIST_DIR)
	go build -ldflags="$(LDFLAGS)" -trimpath -o $(DIST_DIR)/$(BINARY_NAME) $(SRC)
	@echo "✓ Built $(DIST_DIR)/$(BINARY_NAME)"

## run: Run directly with go run
run:
	go run $(SRC)

## deps: Download dependencies and tidy go.mod
deps:
	go mod download
	go mod tidy

## linux: Cross-compile for Linux (amd64 & arm64)
linux: linux-amd64 linux-arm64

linux-amd64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(SRC)
	@echo "✓ Built $(DIST_DIR)/$(BINARY_NAME)-linux-amd64"

linux-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(SRC)
	@echo "✓ Built $(DIST_DIR)/$(BINARY_NAME)-linux-arm64"

## macos: Cross-compile for macOS (Intel & Apple Silicon)
macos: darwin-amd64 darwin-arm64

darwin-amd64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(SRC)
	@echo "✓ Built $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64"

darwin-arm64:
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -trimpath -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(SRC)
	@echo "✓ Built $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64"

## build-all: Build for all Linux and macOS architectures
build-all: linux macos

## clean: Remove build artifacts
clean:
	rm -rf $(DIST_DIR)
	@echo "✓ Removed $(DIST_DIR)"
