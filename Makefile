.PHONY: build run clean test dev-server web-build

# Build frontend
web-build:
	cd web && npm install && npm run build

# Build Go con frontend embebido (copia web/dist → static/dist para //go:embed)
build: web-build
	rm -rf static/dist && cp -r web/dist static/
	go build -o bin/pewpew ./cmd/pewpew

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
