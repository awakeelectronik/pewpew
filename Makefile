.PHONY: build run clean test dev-server web-build

# Build frontend
web-build:
	cd web && npm install && npm run build

# Build Go con embed de web
build: web-build
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
