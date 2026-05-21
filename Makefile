BINARY  := notification-service
BIN_DIR := bin

.PHONY: all build test mocks clean

all: build

## build: compile the binary to bin/
build:
	go build -o $(BIN_DIR)/$(BINARY) ./cmd

## test: run all tests with race detector and coverage
test:
	go test -race -cover ./...

## mocks: (re-)generate mocks from interfaces defined in .mockery.yaml
mocks:
	mockery

## clean: remove compiled binary
clean:
	rm -f $(BIN_DIR)/$(BINARY)
