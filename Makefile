.PHONY: all build clean test run install-deps build-ui

BINARY_NAME=spectre
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags "-s -w"

all: build

install-deps:
	$(GO) mod download
	$(GO) mod verify

build-ui:
	cd web/ui && npm install && npm run build
	rm -rf internal/web/dist/*
	cp -r web/ui/dist/* internal/web/dist/

build: install-deps
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) cmd/spectre-proxy/main.go

build-full: install-deps build-ui
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_NAME) cmd/spectre-proxy/main.go

clean:
	rm -f $(BINARY_NAME)
	rm -f *.db *.db-shm *.db-wal
	rm -rf web/ui/dist web/ui/node_modules
	# Reset internal/web/dist to placeholder only (keep index.html)
	find internal/web/dist -type f ! -name 'index.html' ! -name '.gitkeep' -delete
	find internal/web/dist -type d -empty -delete

clean-all: clean
	# Complete cleanup including placeholder
	rm -rf internal/web/dist
	mkdir -p internal/web/dist

test:
	$(GO) test -v -race -cover ./...

test-coverage:
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

run: build
	./$(BINARY_NAME)

lint:
	golangci-lint run

fmt:
	$(GO) fmt ./...
	$(GO) vet ./...

.DEFAULT_GOAL := build
