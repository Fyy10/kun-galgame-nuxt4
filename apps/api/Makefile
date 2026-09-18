.PHONY: dev build run lint migrate migrate-down migrate-status clean tools openapi

# Development with hot reload (requires air: go install github.com/air-verse/air@latest)
dev:
	air

# Build production binary
build:
	go build -o build/server ./cmd/server

# Run production binary
run: build
	./build/server

# Same gate CI runs: vet plus errcheck (a discarded gorm error is a silent
# no-op write, which vet does not see). Exclusions live in .errcheck-exclude.
lint:
	go vet ./...
	go run github.com/kisielk/errcheck@v1.9.0 -exclude .errcheck-exclude ./...

openapi:
	go run ./cmd/openapi

# Run all pending up migrations
migrate:
	go run ./cmd/migrate -dir up

# Roll back one migration
migrate-down:
	go run ./cmd/migrate -dir down -step 1

# Roll back all migrations
migrate-down-all:
	go run ./cmd/migrate -dir down

# Clean build artifacts
clean:
	rm -rf build/ tmp/

# Install development tools
tools:
	go install github.com/air-verse/air@latest
