.PHONY: build run clean test

build:
	go build -o bin/pewpew ./cmd/pewpew

run:
	go run ./cmd/pewpew/main.go start

clean:
	rm -rf bin/

test:
	go test ./...

lint:
	golangci-lint run ./...
