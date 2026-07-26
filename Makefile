.PHONY: build run test clean sqlc deps

BINARY_NAME=goq
BUILD_DIR=build
MAIN_FILE=cmd/main.go

build:
	go fmt .
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)

run:
	go run $(MAIN_FILE)

sqlc:
	sqlc generate

deps:
	go mod tidy

fmt:
	go fmt .