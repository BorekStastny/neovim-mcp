.PHONY: build clean test install

BINARY_NAME=neovim-mcp
BUILD_DIR=build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

clean:
	rm -rf $(BUILD_DIR)
	go clean

test:
	go test ./...

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

deps:
	go mod download
	go mod tidy

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

dev:
	go run .
