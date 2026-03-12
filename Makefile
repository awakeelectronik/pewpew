.PHONY: build run clean test dev-server web-build release release-linux-amd64 release-linux-arm64

# Build frontend
web-build:
	cd web && npm install && npm run build

# Build Go con frontend embebido (copia web/dist → static/dist para //go:embed)
build: web-build
	rm -rf static/dist && cp -r web/dist static/
	go build -o bin/pewpew ./cmd/pewpew

# Release: binarios linux-amd64 + linux-arm64 + checksums (para GitHub Releases)
release-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/pewpew-linux-amd64 ./cmd/pewpew

release-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/pewpew-linux-arm64 ./cmd/pewpew

release: web-build
	rm -rf static/dist && cp -r web/dist static/
	$(MAKE) release-linux-amd64 release-linux-arm64
	cd bin && sha256sum pewpew-linux-amd64 pewpew-linux-arm64 > checksums.txt

# Desarrollo (frontend + backend separados)
dev-frontend:
	cd web && npm run dev

dev-backend:
	go run ./cmd/pewpew/main.go start

# Run (production)
run: build
	./bin/pewpew start

clean:
	rm -rf bin/ web/dist/

test:
	go test ./...

lint:
	golangci-lint run ./...
