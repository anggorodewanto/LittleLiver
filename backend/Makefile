BINARY := server
BUILD_DIR := bin

.PHONY: all build test vet lint clean

all: vet test build

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/server

test:
	go test ./... -v -cover

vet:
	go vet ./...

lint: vet

clean:
	rm -rf $(BUILD_DIR)
